package receiptprompt

import (
	"math"
	"testing"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
)

func TestParseMoney(t *testing.T) {
	cases := map[string]float64{
		"18564.00":  18564,
		"18.564":    18564,
		"18.564,00": 18564,
		"1,234.56":  1234.56,
		"Rp 3640":   3640,
		"":          0,
		"0":         0,
	}
	for in, want := range cases {
		got := ParseMoney(in)
		if math.Abs(got-want) > 0.001 {
			t.Fatalf("ParseMoney(%q)=%v want %v", in, got, want)
		}
	}
}

func TestNormalizeCaseA_Padang(t *testing.T) {
	r := &domain.SplitbillResult{
		Items: []domain.Item{
			{Name: "Nasi", Price: "50000.00", Quantity: "2", Total: "100000.00"},
			{Name: "Ayam", Price: "41000.00", Quantity: "2", Total: "82000.00"},
		},
		Totals: domain.Totals{
			Subtotal: "182000.00",
			Discount: "0",
			Fees: []domain.Fee{
				{Type: "service_charge", Name: "Service Charge", Amount: "3640.00"},
				{Type: "tax", Name: "PB1", Amount: "18554.00"}, // OCR off by 10
			},
			Tax:     domain.Tax{Name: "PBB", Amount: "18554.00", TotalTax: "18554.00", ServiceCharge: "3640"},
			Total:   "204204.00",
			Payment: "",
		},
	}
	Normalize(r)

	if r.Totals.Fees[1].Amount != "18564.00" {
		t.Fatalf("tax fee amount=%s want 18564.00", r.Totals.Fees[1].Amount)
	}
	if r.Totals.Tax.Amount != "18564.00" || r.Totals.Tax.TotalTax != "18564.00" {
		t.Fatalf("legacy tax=%s/%s", r.Totals.Tax.Amount, r.Totals.Tax.TotalTax)
	}
	if r.Totals.Tax.Name != "PB1" {
		t.Fatalf("tax name=%s want PB1", r.Totals.Tax.Name)
	}
	if r.Totals.ServiceCharge != "3640.00" || r.Totals.Tax.ServiceCharge != "3640.00" {
		t.Fatalf("service_charge legacy=%s nested=%s", r.Totals.ServiceCharge, r.Totals.Tax.ServiceCharge)
	}
	if r.Totals.Payment != "204204.00" {
		t.Fatalf("payment=%s", r.Totals.Payment)
	}
	assertBalance(t, r)
}

func TestNormalizeCaseB_MultiTax(t *testing.T) {
	r := &domain.SplitbillResult{
		Items: []domain.Item{{Name: "X", Price: "100000.00", Quantity: "1", Total: "100000.00"}},
		Totals: domain.Totals{
			Subtotal: "100000.00",
			Discount: "0.00",
			Fees: []domain.Fee{
				{Type: "service_charge", Name: "Service 5%", Amount: "5000.00", Rate: strPtr("5")},
				{Type: "tax", Name: "PB1", Amount: "10000.00", Rate: strPtr("10")},
				{Type: "tax", Name: "PPN", Amount: "11000.00", Rate: strPtr("11")},
			},
			Total: "126000.00",
		},
	}
	Normalize(r)
	if r.Totals.Tax.Amount != "21000.00" || r.Totals.Tax.TotalTax != "21000.00" {
		t.Fatalf("legacy tax sum=%s", r.Totals.Tax.Amount)
	}
	if r.Totals.Tax.Name != "PPN" {
		t.Fatalf("primary tax name=%s want PPN", r.Totals.Tax.Name)
	}
	if r.Totals.ServiceCharge != "5000.00" {
		t.Fatalf("service=%s", r.Totals.ServiceCharge)
	}
	if len(r.Totals.Fees) != 3 {
		t.Fatalf("fees len=%d", len(r.Totals.Fees))
	}
	assertBalance(t, r)
}

func TestNormalizeCaseC_Tip(t *testing.T) {
	r := &domain.SplitbillResult{
		Items: []domain.Item{{Name: "X", Price: "182000.00", Quantity: "1", Total: "182000.00"}},
		Totals: domain.Totals{
			Subtotal: "182000.00",
			Discount: "0.00",
			Fees: []domain.Fee{
				{Type: "service_charge", Name: "Service Charge", Amount: "3640.00"},
				{Type: "tax", Name: "PB1", Amount: "18564.00"},
				{Type: "tip", Name: "Tip", Amount: "5000.00"},
			},
			Total: "209204.00",
		},
	}
	Normalize(r)
	if r.Totals.Total != "209204.00" || r.Totals.Payment != "209204.00" {
		t.Fatalf("total/payment=%s/%s", r.Totals.Total, r.Totals.Payment)
	}
	if r.Totals.Tax.Amount != "18564.00" || r.Totals.ServiceCharge != "3640.00" {
		t.Fatalf("legacy tax=%s sc=%s", r.Totals.Tax.Amount, r.Totals.ServiceCharge)
	}
	assertBalance(t, r)
}

func TestNormalizeCaseD_ManyFees(t *testing.T) {
	fees := make([]domain.Fee, 0, 10)
	var sum float64
	for i := 0; i < 10; i++ {
		fees = append(fees, domain.Fee{Type: "fee", Name: "Packing", Amount: "100.00"})
		sum += 100
	}
	r := &domain.SplitbillResult{
		Items: []domain.Item{{Name: "A", Price: "1000.00", Quantity: "1", Total: "1000.00"}},
		Totals: domain.Totals{
			Subtotal: "1000.00",
			Discount: "0.00",
			Fees:     fees,
			Total:    formatFloat(1000 + sum),
		},
	}
	Normalize(r)
	if len(r.Totals.Fees) != 10 {
		t.Fatalf("fees truncated to %d", len(r.Totals.Fees))
	}
	assertBalance(t, r)
}

func TestNormalizeCaseE_NoFees(t *testing.T) {
	r := &domain.SplitbillResult{
		Items: []domain.Item{{Name: "A", Price: "50000.00", Quantity: "1", Total: "50000.00"}},
		Totals: domain.Totals{
			Subtotal: "50000.00",
			Discount: "0.00",
			Fees:     nil,
			Total:    "50000.00",
		},
	}
	Normalize(r)
	if r.Totals.Fees == nil || len(r.Totals.Fees) != 0 {
		t.Fatalf("fees=%v want empty slice", r.Totals.Fees)
	}
	if r.Totals.Tax.Amount != "0.00" || r.Totals.ServiceCharge != "0.00" {
		t.Fatalf("legacy tax=%s sc=%s", r.Totals.Tax.Amount, r.Totals.ServiceCharge)
	}
	assertBalance(t, r)
}

func TestNormalizeSeedFromLegacy(t *testing.T) {
	r := &domain.SplitbillResult{
		Items: []domain.Item{{Name: "A", Price: "100000.00", Quantity: "1", Total: "100000.00"}},
		Totals: domain.Totals{
			Subtotal: "100000.00",
			Discount: "0.00",
			Fees:     nil,
			Tax: domain.Tax{
				Name:          "PB1",
				Amount:        "10000.00",
				TotalTax:      "10000.00",
				ServiceCharge: "5000.00",
			},
			ServiceCharge: "5000.00",
			Total:         "115000.00",
		},
	}
	Normalize(r)
	if len(r.Totals.Fees) != 2 {
		t.Fatalf("seeded fees=%d want 2", len(r.Totals.Fees))
	}
	assertBalance(t, r)
}

func TestClassifyFeeType(t *testing.T) {
	cases := map[string]string{
		"PB1 10%":          "tax",
		"PPN":              "tax",
		"Service Charge":   "service_charge",
		"Tip":              "tip",
		"Packing Fee":      "fee",
		"Delivery / Ongkir": "fee",
		"Mystery":          "other",
	}
	for name, want := range cases {
		if got := classifyFeeType(name, ""); got != want {
			t.Fatalf("classify(%q)=%s want %s", name, got, want)
		}
	}
}

func assertBalance(t *testing.T, r *domain.SplitbillResult) {
	t.Helper()
	var itemsSum float64
	for _, it := range r.Items {
		itemsSum += ParseMoney(it.Total)
	}
	sub := ParseMoney(r.Totals.Subtotal)
	if math.Abs(itemsSum-sub) > moneyTol {
		t.Fatalf("items sum %v != subtotal %v", itemsSum, sub)
	}
	var feesSum float64
	var taxSum, scSum float64
	for _, f := range r.Totals.Fees {
		amt := ParseMoney(f.Amount)
		feesSum += amt
		switch f.Type {
		case "tax":
			taxSum += amt
		case "service_charge":
			scSum += amt
		}
	}
	computed := sub - ParseMoney(r.Totals.Discount) + feesSum
	total := ParseMoney(r.Totals.Total)
	if math.Abs(computed-total) > moneyTol {
		t.Fatalf("balance %v != total %v", computed, total)
	}
	if math.Abs(taxSum-ParseMoney(r.Totals.Tax.Amount)) > moneyTol {
		t.Fatalf("tax legacy %s != sum %v", r.Totals.Tax.Amount, taxSum)
	}
	if math.Abs(taxSum-ParseMoney(r.Totals.Tax.TotalTax)) > moneyTol {
		t.Fatalf("total_tax legacy mismatch")
	}
	if math.Abs(scSum-ParseMoney(r.Totals.ServiceCharge)) > moneyTol {
		t.Fatalf("service_charge legacy mismatch")
	}
	if math.Abs(scSum-ParseMoney(r.Totals.Tax.ServiceCharge)) > moneyTol {
		t.Fatalf("tax.service_charge legacy mismatch")
	}
}

func strPtr(s string) *string { return &s }
