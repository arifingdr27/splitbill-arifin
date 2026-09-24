package receiptprompt

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
)

const moneyTol = 0.51

var nonMoneyRe = regexp.MustCompile(`[^\d.,\-]`)

// Normalize formats amounts, ensures fees[], derives legacy fields, and enforces balance invariants.
func Normalize(r *domain.SplitbillResult) {
	if r == nil {
		return
	}

	for i := range r.Items {
		r.Items[i].Price = FormatMoney(r.Items[i].Price)
		r.Items[i].Total = FormatMoney(r.Items[i].Total)
		r.Items[i].Quantity = strings.TrimSpace(r.Items[i].Quantity)
	}

	r.StoreInformation.Email = emptyToNil(r.StoreInformation.Email)
	r.StoreInformation.NPWP = emptyToNil(r.StoreInformation.NPWP)
	r.StoreInformation.PhoneNumber = emptyToNil(r.StoreInformation.PhoneNumber)
	r.TransactionInfo.Time = emptyToNil(r.TransactionInfo.Time)
	r.TransactionInfo.TransactionID = emptyToNil(r.TransactionInfo.TransactionID)

	t := &r.Totals
	t.Subtotal = FormatMoney(t.Subtotal)
	t.Discount = FormatMoney(defaultZero(t.Discount))
	t.Total = FormatMoney(t.Total)
	t.Payment = FormatMoney(t.Payment)
	t.ServiceCharge = FormatMoney(t.ServiceCharge)
	t.Change = emptyToNilPtr(t.Change)
	t.Tax.Amount = FormatMoney(t.Tax.Amount)
	t.Tax.TotalTax = FormatMoney(t.Tax.TotalTax)
	t.Tax.ServiceCharge = FormatMoney(t.Tax.ServiceCharge)
	t.Tax.DPP = emptyToNil(t.Tax.DPP)
	t.Tax.Name = normalizeTaxLabel(strings.TrimSpace(t.Tax.Name))

	t.Fees = normalizeFees(t.Fees)
	seedFeesFromLegacy(t)

	for i := range t.Fees {
		t.Fees[i].Type = classifyFeeType(t.Fees[i].Name, t.Fees[i].Type)
		t.Fees[i].Name = normalizeTaxLabel(strings.TrimSpace(t.Fees[i].Name))
		t.Fees[i].Amount = FormatMoney(t.Fees[i].Amount)
		t.Fees[i].Rate = normalizeRate(t.Fees[i].Rate)
	}

	alignSubtotalWithItems(r)
	enforceTotalBalance(t)
	deriveLegacyFromFees(t)

	if ParseMoney(t.Payment) <= 0 {
		t.Payment = t.Total
	}
	t.Change = emptyToNilPtr(t.Change)
}

func normalizeFees(fees []domain.Fee) []domain.Fee {
	if fees == nil {
		return []domain.Fee{}
	}
	out := make([]domain.Fee, 0, len(fees))
	for _, f := range fees {
		if strings.TrimSpace(f.Name) == "" && ParseMoney(f.Amount) == 0 {
			continue
		}
		out = append(out, f)
	}
	return out
}

func seedFeesFromLegacy(t *domain.Totals) {
	if len(t.Fees) > 0 {
		return
	}
	sc := ParseMoney(t.ServiceCharge)
	if sc <= 0 {
		sc = ParseMoney(t.Tax.ServiceCharge)
	}
	if sc > 0 {
		t.Fees = append(t.Fees, domain.Fee{
			Type:   "service_charge",
			Name:   "Service Charge",
			Amount: FormatMoney(strconv.FormatFloat(sc, 'f', -1, 64)),
		})
	}
	taxAmt := ParseMoney(t.Tax.Amount)
	if taxAmt <= 0 {
		taxAmt = ParseMoney(t.Tax.TotalTax)
	}
	if taxAmt > 0 {
		name := t.Tax.Name
		if name == "" {
			name = "PB1"
		}
		t.Fees = append(t.Fees, domain.Fee{
			Type:   "tax",
			Name:   name,
			Amount: FormatMoney(strconv.FormatFloat(taxAmt, 'f', -1, 64)),
		})
	}
}

func alignSubtotalWithItems(r *domain.SplitbillResult) {
	var sum float64
	for _, it := range r.Items {
		sum += ParseMoney(it.Total)
	}
	sub := ParseMoney(r.Totals.Subtotal)
	if len(r.Items) == 0 {
		return
	}
	if sub <= 0 || math.Abs(sum-sub) > moneyTol {
		r.Totals.Subtotal = formatFloat(sum)
	}
}

func enforceTotalBalance(t *domain.Totals) {
	sub := ParseMoney(t.Subtotal)
	disc := ParseMoney(t.Discount)
	feesSum := sumFeeAmounts(t.Fees)
	computed := sub - disc + feesSum
	total := ParseMoney(t.Total)

	if total <= 0 && computed > 0 {
		t.Total = formatFloat(computed)
		return
	}
	delta := total - computed
	if math.Abs(delta) <= moneyTol {
		return
	}
	if len(t.Fees) == 0 {
		t.Total = formatFloat(computed)
		return
	}
	idx := pickAdjustFeeIndex(t.Fees)
	amt := ParseMoney(t.Fees[idx].Amount) + delta
	if amt < 0 {
		t.Total = formatFloat(computed)
		return
	}
	t.Fees[idx].Amount = formatFloat(amt)
}

func pickAdjustFeeIndex(fees []domain.Fee) int {
	bestTax := -1
	bestTaxAmt := -1.0
	bestAny := 0
	bestAnyAmt := -1.0
	for i, f := range fees {
		amt := ParseMoney(f.Amount)
		if amt >= bestAnyAmt {
			bestAnyAmt = amt
			bestAny = i
		}
		if f.Type == "tax" && amt >= bestTaxAmt {
			bestTaxAmt = amt
			bestTax = i
		}
	}
	if bestTax >= 0 {
		return bestTax
	}
	return bestAny
}

func deriveLegacyFromFees(t *domain.Totals) {
	var taxSum, scSum float64
	var primaryName string
	var primaryAmt float64
	firstTax := true
	for _, f := range t.Fees {
		amt := ParseMoney(f.Amount)
		switch f.Type {
		case "tax":
			taxSum += amt
			if firstTax || amt > primaryAmt {
				primaryName = f.Name
				primaryAmt = amt
				firstTax = false
			}
		case "service_charge":
			scSum += amt
		}
	}
	t.Tax.Amount = formatFloat(taxSum)
	t.Tax.TotalTax = formatFloat(taxSum)
	if primaryName != "" {
		t.Tax.Name = primaryName
	} else if t.Tax.Name == "" {
		t.Tax.Name = ""
	}
	sc := formatFloat(scSum)
	t.ServiceCharge = sc
	t.Tax.ServiceCharge = sc
}

func sumFeeAmounts(fees []domain.Fee) float64 {
	var s float64
	for _, f := range fees {
		s += ParseMoney(f.Amount)
	}
	return s
}

func classifyFeeType(name, typ string) string {
	typ = strings.ToLower(strings.TrimSpace(typ))
	switch typ {
	case "tax", "service_charge", "tip", "fee", "other":
		if typ != "other" {
			return typ
		}
	}
	n := strings.ToLower(name)
	switch {
	case containsAny(n, "pb1", "ppn", "ppnbm", "vat", "pajak") || hasWord(n, "tax"):
		return "tax"
	case containsAny(n, "service charge", "service", "svc") || hasWord(n, "sc"):
		return "service_charge"
	case containsAny(n, "gratuity", "tips") || hasWord(n, "tip"):
		return "tip"
	case containsAny(n, "packing", "delivery", "ongkir", "takeaway", "kemasan", "round"):
		return "fee"
	case typ == "other":
		return "other"
	default:
		return "other"
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func hasWord(s, word string) bool {
	for _, p := range strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if strings.EqualFold(p, word) {
			return true
		}
	}
	return false
}

func normalizeTaxLabel(name string) string {
	if strings.EqualFold(strings.TrimSpace(name), "PBB") {
		return "PB1"
	}
	return name
}

func normalizeRate(r *string) *string {
	if r == nil {
		return nil
	}
	s := strings.TrimSpace(*r)
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func emptyToNil(p *string) *string {
	if p == nil {
		return nil
	}
	if strings.TrimSpace(*p) == "" {
		return nil
	}
	return p
}

func emptyToNilPtr(p *string) *string {
	return emptyToNil(p)
}

func defaultZero(s string) string {
	if strings.TrimSpace(s) == "" {
		return "0.00"
	}
	return s
}

// FormatMoney returns a plain "N.NN" string.
func FormatMoney(s string) string {
	return formatFloat(ParseMoney(s))
}

func formatFloat(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0.00"
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// ParseMoney parses OCR money strings including ID/EU thousand separators.
func ParseMoney(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = nonMoneyRe.ReplaceAllString(s, "")
	if s == "" || s == "-" || s == "." || s == "," {
		return 0
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = strings.TrimPrefix(s, "-")
	}

	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")

	var normalized string
	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastComma > lastDot {
			// 1.234,56
			normalized = strings.ReplaceAll(s[:lastComma], ".", "")
			normalized = strings.ReplaceAll(normalized, ",", "")
			normalized += "." + s[lastComma+1:]
		} else {
			// 1,234.56
			normalized = strings.ReplaceAll(s[:lastDot], ",", "")
			normalized = strings.ReplaceAll(normalized, ".", "")
			normalized += "." + s[lastDot+1:]
		}
	case lastComma >= 0:
		parts := strings.Split(s, ",")
		if len(parts) == 2 && len(parts[1]) > 0 && len(parts[1]) <= 2 {
			normalized = parts[0] + "." + parts[1]
		} else {
			normalized = strings.ReplaceAll(s, ",", "")
		}
	case lastDot >= 0:
		parts := strings.Split(s, ".")
		if looksLikeThousandDots(parts) {
			normalized = strings.ReplaceAll(s, ".", "")
		} else if len(parts) == 2 {
			normalized = s
		} else {
			normalized = strings.ReplaceAll(s, ".", "")
		}
	default:
		normalized = s
	}

	v, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0
	}
	if neg {
		v = -v
	}
	return v
}

func looksLikeThousandDots(parts []string) bool {
	if len(parts) < 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[0]) > 3 {
		return false
	}
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) != 3 {
			return false
		}
	}
	return true
}
