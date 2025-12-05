package protocolfactory

import "testing"

func TestServiceFactory(t *testing.T) {
    f := New()
    if f == nil {
        t.Fatal("expected non-nil factory")
    }
}
