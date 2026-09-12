package receiptprompt

// Extract is the shared OCR instruction for all vision providers.
const Extract = `Lakukan Optical Character Recognition (OCR) pada gambar struk ini dan ekstrak informasi belanja.
Kembalikan HANYA JSON valid sesuai schema, tanpa markdown, tanpa penjelasan, tanpa teks di luar JSON.

Aturan field:
- items: daftar barang dari struk
- price: harga per unit; jika tidak ada kolom terpisah, hitung total/quantity; jika tidak bisa, "0"
- quantity, total: sesuai struk
- nilai numerik: desimal tanpa pemisah ribuan (contoh "220000.00")
- field tidak ditemukan: string kosong ""
- date: DD/MM/YYYY
- time: HH:MM
- discount: angka desimal; jika tidak ada, "0"

Schema JSON wajib:
{
  "items": [{"name":"","price":"","quantity":"","total":""}],
  "store_information": {"address":"","email":"","npwp":"","phone_number":"","store_name":""},
  "totals": {
    "change":"","discount":"","payment":"","subtotal":"",
    "tax":{"amount":"","service_charge":"","dpp":"","name":"","total_tax":""},
    "total":""
  },
  "transaction_information": {"date":"","time":"","transaction_id":""}
}`
