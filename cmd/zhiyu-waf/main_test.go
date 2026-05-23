package main

import "testing"

func TestGenerateOTPUsesCommercialStrengthLength(t *testing.T) {
	otp := generateOTP()
	if len(otp) != 24 {
		t.Fatalf("expected 24-character initial password, got %d", len(otp))
	}
	if otp == generateOTP() {
		t.Fatal("generated initial passwords should not repeat")
	}
}
