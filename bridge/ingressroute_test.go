package main

import (
	"reflect"
	"testing"
)

func TestHostMatchExpr(t *testing.T) {
	got := hostMatchExpr([]string{"a.example.com", "b.example.com", "a.example.com", ""})
	want := `Host("a.example.com") || Host("b.example.com")`
	if got != want {
		t.Errorf("hostMatchExpr() = %q, want %q", got, want)
	}
}

func TestAcmeDomains(t *testing.T) {
	t.Run("skips the canonical (first) host", func(t *testing.T) {
		got := acmeDomains([]string{"ws-abc.hermeshq.net", "ws-abc.runtime-kh-test.hermeshq.net"})
		want := []any{map[string]any{"main": "ws-abc.runtime-kh-test.hermeshq.net"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("acmeDomains() = %#v, want %#v", got, want)
		}
	})

	t.Run("no additional hosts means no domains to validate", func(t *testing.T) {
		got := acmeDomains([]string{"ws-abc.hermeshq.net"})
		if len(got) != 0 {
			t.Errorf("acmeDomains() = %#v, want empty", got)
		}
	})

	t.Run("skips blank additional hosts", func(t *testing.T) {
		got := acmeDomains([]string{"ws-abc.hermeshq.net", "  ", "ws-abc.runtime-kh-test.hermeshq.net"})
		want := []any{map[string]any{"main": "ws-abc.runtime-kh-test.hermeshq.net"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("acmeDomains() = %#v, want %#v", got, want)
		}
	})
}
