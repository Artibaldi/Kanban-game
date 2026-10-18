package game

import "testing"

func TestTavern(t *testing.T) {
	d, ok := Buildings["tavern"]
	if !ok || d.Name != "Таверна" || d.Defense {
		t.Fatalf("tavern: %+v", d)
	}
	if d.Allowed(1) != 1 || d.Allowed(5) != 1 || len(d.Levels) != 3 {
		t.Fatalf("one tavern, three levels: %+v", d)
	}
	for i, l := range d.Levels {
		if l.RequiresTH != i+1 || l.Cost <= 0 || l.HP <= 0 {
			t.Fatalf("level %d: %+v", i+1, l)
		}
	}
}

func TestFriendlyLimit(t *testing.T) {
	s := DefaultSettings()
	for tavern, want := range map[int]int{0: 3, 1: 4, 2: 5, 3: 5} {
		if got := FriendlyLimit(s, tavern); got != want {
			t.Errorf("tavern %d: %d, want %d", tavern, got, want)
		}
	}
	if ChallengeTTL.Hours() != 24 {
		t.Fatalf("a challenge lives 24 hours: %v", ChallengeTTL)
	}
}
