package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"zhiyuwaf/internal/ai"
	"zhiyuwaf/internal/ai/openai"
	"zhiyuwaf/internal/acme"
	"zhiyuwaf/internal/alert"
	"zhiyuwaf/internal/config"
	"zhiyuwaf/internal/dashboard"
	"zhiyuwaf/internal/engine"
	"zhiyuwaf/internal/geo"
	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/proxy"
	"zhiyuwaf/internal/sshmon"
	"zhiyuwaf/internal/store"
	"zhiyuwaf/internal/threatintel"
)

func main() {
	configPath := flag.String("config", "configs/zhiyu-waf.yaml", "path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}
	log.Printf("config loaded from %s", *configPath)

	applyEnvOverrides(cfg)

	// Initialize store
	var dbStore store.Storage
	switch cfg.Storage.Type {
	case "mysql":
		if cfg.Storage.DSN == "" {
			log.Fatal("MySQL selected but storage.dsn is empty")
		}
		dbStore, err = store.NewMySQLStore(cfg.Storage.DSN, cfg.Storage.MaxOpenConns, cfg.Storage.MaxIdleConns)
	default:
		dbStore, err = store.NewStore(cfg.Storage.Path)
	}
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}
	defer dbStore.Close()
	log.Printf("database initialized at %s", cfg.Storage.Path)

	// Init geo rules table
	if err := dbStore.InitGeoTable(); err != nil {
		log.Printf("warning: failed to init geo table: %v", err)
	}

	// First-time setup: generate one-time password if none set
	if storedHash, _ := dbStore.GetSetting("admin_password_hash"); storedHash == "" {
		otp := generateOTP()
		hash, _ := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
		dbStore.SetSetting("admin_password_hash", string(hash))
		log.Printf("=======================================================")
		log.Printf("首次启动 - 管理员初始密码: %s", otp)
		log.Printf("请登录后立即修改密码！")
		log.Printf("=======================================================")
	}

	// Initialize GeoIP resolver
	geoResolver := geo.NewResolver()

	// Initialize SSH monitor
	sshMonitor := sshmon.New(sshmon.Config{
		Enabled:         cfg.SSH.Enabled,
		LogPath:         cfg.SSH.LogPath,
		MaxFails:        cfg.SSH.MaxFails,
		BanMinutes:      cfg.SSH.BanMinutes,
		IPTablesEnabled: cfg.Proxy.IPTablesEnable,
	}, dbStore, geoResolver)
	sshMonitor.Start()
	defer sshMonitor.Stop()

	// Create dashboard server and load persisted AI settings
	dashServer := dashboard.NewServer(cfg, dbStore)
	dashServer.SetConfigPath(*configPath)
	dashServer.LoadAISettingsFromDB()
	applyEnvOverrides(cfg)

	// Load rules from YAML files
	ruleSet := engine.NewRuleSet()
	ruleSet.SetPreset(cfg.Engine.Preset)
	if err := ruleSet.LoadFromDir(cfg.Engine.RulesDir); err != nil {
		log.Fatalf("failed to load rules: %v", err)
	}

	// Also load rules from DB
	dbRules, err := dbStore.ListRules()
	if err != nil {
		log.Printf("warning: failed to load DB rules: %v", err)
	} else {
		ruleSet.LoadFromDB(dbRules)
		log.Printf("loaded %d rules from database", len(dbRules))
	}

	// Create detection pipeline
	pipeline := engine.NewPipeline(ruleSet, cfg.Engine.RateLimit.RequestsPerMinute, cfg.Engine.RateLimit.BurstSize)
	pipeline.SetObservationMode(cfg.Engine.ObservationMode)
	if cfg.Engine.ObservationMode {
		log.Println("WARNING: observation mode enabled — requests will NOT be blocked, only logged")
	}

	// Load IP lists
	whitelist, _ := dbStore.GetIPListMap("whitelist")
	blacklist, _ := dbStore.GetIPListMap("blacklist")
	pipeline.UpdateIPLists(whitelist, blacklist)

	// Set geo resolver for geo-blocking
	pipeline.SetGeoResolver(geoResolver)
	if blocked, err := dbStore.GetBlockedCountries(); err == nil {
		pipeline.UpdateGeoRules(blocked)
		log.Printf("loaded %d geo-blocked countries", len(blocked))
	}

	// Initialize AI analyzer (if enabled)
	var currentAI ai.Analyzer
	currentAI = initAI(cfg, pipeline, dashServer.IncrementAIUsage)
	if currentAI != nil {
		currentAI.SetAllowedCheck(dashServer.IsCommunityAIAllowed)
	}

	var handler *proxy.Handler
	siteResolver := proxy.NewMemorySiteResolver(loadEnabledSites(dbStore))

	// Wire up callbacks for hot-reload
	dashServer.OnAIConfigChanged = func() {
		log.Println("AI config changed, reinitializing analyzer...")
		if currentAI != nil {
			currentAI.Stop()
		}
		currentAI = initAI(cfg, pipeline, dashServer.IncrementAIUsage)
		if currentAI != nil {
			currentAI.SetAllowedCheck(dashServer.IsCommunityAIAllowed)
		}
	}
	dashServer.OnConfigReload = func() {
		log.Println("config reload requested")
		newCfg, err := config.Load(*configPath)
		if err != nil {
			log.Printf("reload config failed: %v", err)
			return
		}
		newRuleSet := engine.NewRuleSet()
		newRuleSet.SetPreset(newCfg.Engine.Preset)
		if err := newRuleSet.LoadFromDir(newCfg.Engine.RulesDir); err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		dbRules, _ := dbStore.ListRules()
		newRuleSet.LoadFromDB(dbRules)
		pipeline.UpdateRules(newRuleSet)
		// Update handler config to avoid stale references
		handler.UpdateConfig(newCfg.Proxy.BackendAddr, newCfg.Proxy.ReadTimeout, newCfg.Proxy.WriteTimeout, newCfg.Proxy.DynamicProtect)
		log.Println("config reloaded")
	}
	dashServer.OnIPListChanged = func() {
		whitelist, _ := dbStore.GetIPListMap("whitelist")
		blacklist, _ := dbStore.GetIPListMap("blacklist")
		pipeline.UpdateIPLists(whitelist, blacklist)
		log.Println("IP lists reloaded")
	}
	dashServer.OnSitesChanged = func() {
		siteResolver.Update(loadEnabledSites(dbStore))
		log.Println("sites reloaded")
	}
	dashServer.OnRulesChanged = func() {
		activeCfg, err := config.Load(*configPath)
		if err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		applyEnvOverrides(activeCfg)
		newRuleSet := engine.NewRuleSet()
		newRuleSet.SetPreset(activeCfg.Engine.Preset)
		if err := newRuleSet.LoadFromDir(activeCfg.Engine.RulesDir); err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		dbRules, _ := dbStore.ListRules()
		newRuleSet.LoadFromDB(dbRules)
		pipeline.UpdateRules(newRuleSet)
		log.Println("rules reloaded")
	}
	dashServer.OnGeoRulesChanged = func() {
		if blocked, err := dbStore.GetBlockedCountries(); err == nil {
			pipeline.UpdateGeoRules(blocked)
			log.Println("geo rules reloaded")
		}
	}

	// Initialize threat intelligence syncer
	var threatSyncer *threatintel.Syncer
	threatSyncer = setupThreatIntel(cfg, dbStore, func() {
		whitelist, _ := dbStore.GetIPListMap("whitelist")
		blacklist, _ := dbStore.GetIPListMap("blacklist")
		pipeline.UpdateIPLists(whitelist, blacklist)
		log.Println("threat intel IPs synced to blacklist")
	})
	if threatSyncer != nil {
		dashServer.ThreatSyncerStatus = func() (time.Time, int) { return threatSyncer.Status() }
		dashServer.ThreatSyncerSync = func() { threatSyncer.Sync() }
	}
	dashServer.OnThreatIntelChanged = func() {
		log.Println("threat intel config changed, reinitializing...")
		if threatSyncer != nil {
			threatSyncer.Stop()
		}
		threatSyncer = setupThreatIntel(cfg, dbStore, func() {
			whitelist, _ := dbStore.GetIPListMap("whitelist")
			blacklist, _ := dbStore.GetIPListMap("blacklist")
			pipeline.UpdateIPLists(whitelist, blacklist)
			log.Println("threat intel IPs synced to blacklist")
		})
		if threatSyncer != nil {
			dashServer.ThreatSyncerStatus = func() (time.Time, int) { return threatSyncer.Status() }
			dashServer.ThreatSyncerSync = func() { threatSyncer.Sync() }
			// Trigger immediate sync after config change
			go threatSyncer.Sync()
		} else {
			dashServer.ThreatSyncerStatus = nil
			dashServer.ThreatSyncerSync = nil
		}
	}

	// Initialize alert channels
	var alerters []alert.Alerter
	if cfg.Alert.Enabled {
		if cfg.Alert.WebhookURL != "" {
			alerters = append(alerters, alert.NewWebhookAlerter(cfg.Alert.WebhookURL, cfg.Alert.ThrottleMin))
			log.Printf("alert webhook configured: %s", cfg.Alert.WebhookURL)
		}
		if cfg.Alert.Email.Host != "" && len(cfg.Alert.Email.To) > 0 {
			alerters = append(alerters, alert.NewEmailAlerter(
				cfg.Alert.Email.Host, cfg.Alert.Email.Port,
				cfg.Alert.Email.Username, cfg.Alert.Email.Password,
				cfg.Alert.Email.From, cfg.Alert.Email.To,
				cfg.Alert.ThrottleMin,
			))
			log.Printf("alert email configured: %s", cfg.Alert.Email.Host)
		}
	}

	// Start background log writer + WebSocket broadcast
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for logEntry := range pipeline.AttackLogChan() {
			// Resolve GeoIP region for the attacker
			logEntry.Region = geoResolver.FormatRegion(logEntry.ClientIP)
			if err := dbStore.InsertAttackLog(logEntry); err != nil {
				log.Printf("failed to save attack log: %v", err)
			}
			if logEntry.Source == "ai" {
				if err := dbStore.InsertAuditEvent(model.AuditEvent{
					ID:        "ai-" + logEntry.ID,
					Timestamp: time.Now(),
					Actor:     "ai",
					ClientIP:  logEntry.ClientIP,
					Action:    "ai_block",
					Status:    "blocked",
					Detail:    logEntry.RuleName + ": " + logEntry.AIReasoning,
				}); err != nil {
					log.Printf("failed to save AI audit event: %v", err)
				}
			}
			// Send alerts for high/critical severity attacks
			if len(alerters) > 0 && (logEntry.Severity == "high" || logEntry.Severity == "critical") {
				a := alert.Alert{
					Title:     "WAF Attack Blocked: " + logEntry.RuleName,
					Severity:  logEntry.Severity,
					Message:   logEntry.Path + " from " + logEntry.ClientIP,
					SourceIP:  logEntry.ClientIP,
					RuleID:    logEntry.RuleID,
					Timestamp: logEntry.Timestamp,
				}
				for _, alerter := range alerters {
					go alerter.Send(a)
				}
			}
			dashServer.Hub().Broadcast(logEntry)
		}
	}()

	// Periodic log cleanup (daily)
	if cfg.Storage.LogRetentionDays > 0 {
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					dbStore.CleanupOldLogs(cfg.Storage.LogRetentionDays)
				case <-ctx.Done():
					return
				}
			}
		}()
		go dbStore.CleanupOldLogs(cfg.Storage.LogRetentionDays)
	}

	// Start config hot-reload (file watcher)
	config.Watch(*configPath, func(newCfg *config.Config) {
		log.Println("config file changed, reloading...")
		newRuleSet := engine.NewRuleSet()
		newRuleSet.SetPreset(newCfg.Engine.Preset)
		if err := newRuleSet.LoadFromDir(newCfg.Engine.RulesDir); err != nil {
			log.Printf("reload rules failed: %v", err)
			return
		}
		dbRules, _ := dbStore.ListRules()
		newRuleSet.LoadFromDB(dbRules)
		pipeline.UpdateRules(newRuleSet)
		// Update handler config to avoid stale references
		handler.UpdateConfig(newCfg.Proxy.BackendAddr, newCfg.Proxy.ReadTimeout, newCfg.Proxy.WriteTimeout, newCfg.Proxy.DynamicProtect)
	})

	// Setup iptables
	// iptables redirects traffic destined for iptablesPort (e.g. 80) to the WAF listen port (e.g. 8080)
	_, wafPort, _ := parseAddr(cfg.Proxy.ListenAddr)
	iptablesMgr := proxy.NewIPTablesManager(wafPort, cfg.Proxy.IPTablesEnable)
	iptablesMgr.SetTLSEnabled(cfg.Proxy.TLSCertFile != "" && cfg.Proxy.TLSKeyFile != "")

	// Log compatibility advice (nginx detection, port conflicts)
	proxy.LogCompatAdvice(wafPort, cfg.Proxy.IPTablesPort)

	if err := iptablesMgr.Setup(cfg.Proxy.IPTablesPort); err != nil {
		log.Printf("warning: iptables setup failed: %v (run as root for iptables support)", err)
	}
	defer iptablesMgr.Cleanup()

	// Create proxy handler and listener
	handler = proxy.NewHandler(cfg.Proxy.BackendAddr, pipeline, cfg.Proxy.ReadTimeout, cfg.Proxy.WriteTimeout)
	handler.SetSiteResolver(siteResolver)
	handler.SetDynamicProtect(cfg.Proxy.DynamicProtect)
	handler.SetMetricsCallbacks(dashboard.IncrementRequests, dashboard.IncrementBlocked)
	listener := proxy.NewListener(cfg.Proxy.ListenAddr, cfg.Proxy.TLSListenAddr, handler, cfg.Proxy.TLSCertFile, cfg.Proxy.TLSKeyFile)

	// Setup ACME if enabled (auto TLS certificates from Let's Encrypt)
	if cfg.Proxy.ACMEEnabled && len(cfg.Proxy.ACMEDomains) > 0 {
		certsDir := filepath.Join(filepath.Dir(cfg.Storage.Path), "certs")
		acmeMgr := acme.New(certsDir, cfg.Proxy.ACMEEmail, cfg.Proxy.ACMEDomains)
		listener.SetACMEManager(acmeMgr)
		log.Printf("ACME enabled for domains: %v", cfg.Proxy.ACMEDomains)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down...", sig)
		cancel()
		if currentAI != nil {
			currentAI.Stop()
		}
		pipeline.Close()
		iptablesMgr.Cleanup()
		os.Exit(0)
	}()

	// Start dashboard in background
	go func() {
		if err := dashServer.Start(ctx); err != nil {
			log.Printf("dashboard error: %v", err)
		}
	}()

	// Start proxy
	log.Printf("ZhiYu-WAF starting on %s -> %s", cfg.Proxy.ListenAddr, cfg.Proxy.BackendAddr)
	log.Printf("Rules loaded: %d", len(ruleSet.Rules()))
	log.Printf("Dashboard at %s", cfg.Dashboard.ListenAddr)

	if err := listener.Start(ctx); err != nil {
		log.Fatalf("proxy failed: %v", err)
	}
}

func applyEnvOverrides(cfg *config.Config) {
	if envSecret := os.Getenv("ZHIYU_WAF_JWT_SECRET"); envSecret != "" {
		cfg.Dashboard.JWTSecret = envSecret
		log.Printf("JWT secret loaded from environment")
	}
	if apiKey := os.Getenv("ZHIYU_WAF_OPENAI_API_KEY"); apiKey != "" {
		cfg.AI.Providers.OpenAI.APIKey = apiKey
		log.Printf("OpenAI-compatible API key loaded from environment")
	}
	if baseURL := os.Getenv("ZHIYU_WAF_OPENAI_BASE_URL"); baseURL != "" {
		cfg.AI.Providers.OpenAI.BaseURL = baseURL
	}
	if model := os.Getenv("ZHIYU_WAF_OPENAI_MODEL"); model != "" {
		cfg.AI.Providers.OpenAI.Model = model
	}
}

// initAI initializes or reinitializes the AI analyzer based on current config.
func initAI(cfg *config.Config, pipeline *engine.Pipeline, onCall func()) ai.Analyzer {
	if !cfg.AI.Enabled {
		pipeline.SetAIAnalyzer(nil)
		log.Println("AI analyzer disabled")
		return nil
	}

	// Guard: empty API key causes continuous failures → circuit breaker → false blocks
	if cfg.AI.Providers.OpenAI.APIKey == "" {
		pipeline.SetAIAnalyzer(nil)
		log.Println("AI analyzer disabled: API key is empty")
		return nil
	}

	var provider ai.Provider
	switch cfg.AI.Provider {
	case "openai":
		provider = openai.NewClient(
			cfg.AI.Providers.OpenAI.APIKey,
			cfg.AI.Providers.OpenAI.Model,
			cfg.AI.Providers.OpenAI.BaseURL,
		)
	default:
		log.Printf("unknown AI provider: %s, AI disabled", cfg.AI.Provider)
		return nil
	}

	analyzer := ai.NewAnalyzer(
		provider,
		time.Duration(cfg.AI.CacheTTL)*time.Second,
		cfg.AI.MaxRequests,
		cfg.AI.AsyncTimeout,
		cfg.AI.FailOpen,
		cfg.AI.HighRiskPaths,
		cfg.AI.PerIPRate,
		cfg.AI.PerIPBurst,
		cfg.AI.CircuitThreshold,
		cfg.AI.CircuitReset,
	)
	if onCall != nil {
		analyzer.SetOnCall(onCall)
	}
	pipeline.SetAIAnalyzer(analyzer)
	log.Printf("AI analyzer initialized: provider=%s model=%s", cfg.AI.Provider, provider.Name())
	return analyzer
}

func parseAddr(addr string) (string, int, error) {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	return addr, port, err
}

func loadEnabledSites(dbStore store.Storage) []model.Site {
	sites, err := dbStore.ListEnabledSites()
	if err != nil {
		log.Printf("warning: failed to load sites: %v", err)
		return nil
	}
	log.Printf("loaded %d enabled sites", len(sites))
	return sites
}

func generateOTP() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b) // 24-char hex password
}

func setupThreatIntel(cfg *config.Config, dbStore store.Storage, onChanged func()) *threatintel.Syncer {
	apiKey := os.Getenv("ZHIYU_WAF_ABUSEIPDB_KEY")
	if apiKey == "" {
		if v, _ := dbStore.GetSetting("threatintel_api_key"); v != "" {
			apiKey = v
		}
	}
	if apiKey == "" {
		log.Println("threat intelligence: no API key configured, skipping")
		return nil
	}

	feed := threatintel.NewAbuseIPDB(apiKey)
	syncer := threatintel.NewSyncer(feed, dbStore, onChanged)
	syncer.Start(6 * time.Hour)
	log.Println("threat intelligence: AbuseIPDB syncer started (every 6h)")
	return syncer
}
