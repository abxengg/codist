package app

import (
	"slices"
	"testing"

	java "github.com/abxengg/codist-java"
)

func TestChatWordsSplitsCamelCase(t *testing.T) {
	got := chatWords("Where does PaymentService retry the charge?")
	for _, w := range []string{"paymentservice", "payment", "service", "retry", "charge"} {
		if !slices.Contains(got, w) {
			t.Errorf("missing %q in %v", w, got)
		}
	}
}

func TestChatPickRanksTheNamedClassFirst(t *testing.T) {
	ix := &java.Index{Types: []*java.Type{
		{ID: "a.OrderService", Name: "OrderService", File: "a/OrderService.java", Methods: []java.Method{{Name: "pay"}}},
		{ID: "a.PaymentService", Name: "PaymentService", File: "a/PaymentService.java", Methods: []java.Method{{Name: "charge"}}},
		{ID: "a.PaymentServiceTest", Name: "PaymentServiceTest", File: "a/PaymentServiceTest.java", Stereotype: "test"},
	}}
	got := chatPick(ix, chatWords("How does PaymentService charge a card?"), "")
	if len(got) == 0 || got[0].Name != "PaymentService" {
		t.Fatalf("want PaymentService first, got %v", got)
	}
	for _, x := range got {
		if x.Stereotype == "test" {
			t.Errorf("tests should not be picked: %s", x.Name)
		}
	}
	// The file on screen wins even when the question does not name it.
	if got := chatPick(ix, chatWords("what does this do"), "a/OrderService.java"); len(got) == 0 || got[0].Name != "OrderService" {
		t.Errorf("want the open file first, got %v", got)
	}
}
