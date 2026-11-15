package config

import "testing"

func TestSupabaseProjectURL(t *testing.T) {
	const want = "https://abc.supabase.co"
	for _, in := range []string{
		"https://abc.supabase.co",
		"https://abc.supabase.co/",
		"https://abc.supabase.co/rest/v1",
		"https://abc.supabase.co/rest/v1/",
		"https://abc.supabase.co/auth/v1",
		"  https://abc.supabase.co/rest/v1/  ",
	} {
		if got := supabaseProjectURL(in); got != want {
			t.Errorf("supabaseProjectURL(%q) = %q, want %q", in, got, want)
		}
	}
	if got := supabaseProjectURL(""); got != "" {
		t.Errorf("empty input gave %q", got)
	}
}
