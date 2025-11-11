package commands

import "testing"

func TestSanitizeGuildName(t *testing.T) {
    cases := []struct{ in, wantPrefix string }{
        {"My Cool Guild", "my-cool-guild"},
        {"  Spaces  ", "spaces"},
        {"$Inv@lid#Chars!", "invlidchars"},
        {"", "guild-"},
    }
    for _, c := range cases {
        got := sanitizeGuildName(c.in)
        if c.in == "" {
            if len(got) == 0 || got[:6] != "guild-" {
                t.Fatalf("expected prefix guild-, got %q", got)
            }
            continue
        }
        if got != c.wantPrefix {
            t.Fatalf("sanitizeGuildName(%q)=%q, want %q", c.in, got, c.wantPrefix)
        }
    }
}

func TestParseDiscordID(t *testing.T) {
    cases := []struct{ in, want string }{
        {"<@123>", "123"},
        {"<@!456>", "456"},
        {"789", "789"},
    }
    for _, c := range cases {
        got := parseDiscordID(c.in)
        if got != c.want {
            t.Fatalf("parseDiscordID(%q)=%q, want %q", c.in, got, c.want)
        }
    }
}
