package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/sirupsen/logrus"
	"google.golang.org/genai"
)

const extractPrompt = `Tolong lakukan Optical Character Recognition (OCR) pada gambar struk ini dan ekstrak informasi belanja. Kembalikan hasilnya dalam format JSON dengan struktur berikut:
{
  "items": [
    {
      "name": "[Nama Barang 1]",
      "price": "[Harga per Unit 1] - Jika tidak tersedia secara eksplisit sebagai kolom terpisah, hitung sebagai [Total Harga Item 1] dibagi [Kuantitas 1]. Jika pembagian menghasilkan angka tidak terbatas (misalnya, total 0 dan kuantitas 0), gunakan 0.",
      "quantity": "[Kuantitas 1]",
      "total": "[Total Harga Item 1]"
    }
  ],
  "store_information": {
    "address": "[Alamat Toko]",
    "email": "[Email Toko]",
    "npwp": "[NPWP Toko]",
    "phone_number": "[Nomor Telepon Toko]",
    "store_name": "[Nama Toko]"
  },
  "totals": {
    "change": "[Uang Kembali]",
    "discount": "[Nilai Diskon/Nilai Yang Dikurangi]. Kembalikan angka desimal tanpa pengurangan. Jika tidak ada diskon, gunakan 0.",
    "payment": "[Jumlah Pembayaran]",
    "subtotal": "[Subtotal]",
    "tax": {
      "amount": "[Nilai Pajak]",
      "service_charge": "[Biaya Layanan]",
      "dpp": "[Dasar Pengenaan Pajak]",
      "name": "[Nama Pajak]",
      "total_tax": "[Total Pajak dari service_charge + amount]"
    },
    "total": "[Total Belanja]"
  },
  "transaction_information": {
    "date": "[Tanggal Transaksi] dalam format DD/MM/YYYY",
    "time": "[Waktu Transaksi] dalam format HH:MM",
    "transaction_id": "[ID Transaksi]"
  }
}

Pastikan semua nilai diisi sesuai dengan informasi yang tertera pada struk. Jika suatu informasi tidak ditemukan, gunakan nilai null atau string kosong untuk field yang sesuai. Untuk nilai numerik (harga, kuantitas, total, totals, discount, dll.), kembalikan dalam format desimal tanpa pemisah ribuan (misalnya, "220000.00" bukan "220,000.00").`

type Extractor struct {
	client *genai.Client
	model  string
	log    *logrus.Logger
}

func NewExtractor(ctx context.Context, apiKey, model string, log *logrus.Logger) (*Extractor, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &Extractor{client: client, model: model, log: log}, nil
}

func (e *Extractor) Extract(ctx context.Context, image []byte, mimeType string) (*domain.SplitbillResult, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	parts := []*genai.Part{
		genai.NewPartFromText(extractPrompt),
		{InlineData: &genai.Blob{MIMEType: mimeType, Data: image}},
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	result, err := e.client.Models.GenerateContent(ctx, e.model, contents, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrExtractFailed, err)
	}

	cleaned := cleanJSON(result.Text())
	e.log.Debugf("gemini raw response cleaned length=%d", len(cleaned))

	var out domain.SplitbillResult
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("%w: unmarshal: %v", domain.ErrExtractFailed, err)
	}
	return &out, nil
}

func cleanJSON(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
