package services

import "testing"

func TestService(t *testing.T) {
    s := NewService()
    if s == nil {
        t.Fatal("expected non-nil service")
    }
}
