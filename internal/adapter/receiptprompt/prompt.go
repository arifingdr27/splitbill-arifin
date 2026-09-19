package receiptprompt

// Extract is the shared OCR instruction for all vision providers.
const Extract = `Lakukan Optical Character Recognition (OCR) pada gambar struk ini dan ekstrak informasi belanja.
Kembalikan HANYA JSON valid sesuai schema, tanpa markdown, tanpa penjelasan, tanpa teks di luar JSON.

Aturan field:
- items: daftar barang dari struk
- price: harga per unit; jika tidak ada kolom terpisah, hitung total/quantity; jika tidak bisa, "0"
- quantity, total: sesuai struk
- nilai numerik: desimal tanpa pemisah ribuan (contoh "220000.00"); JANGAN sisipkan simbol atau kode mata uang ke field numerik
- field tidak ditemukan: string kosong ""
- date: DD/MM/YYYY
- time: HH:MM
- discount: angka desimal; jika tidak ada, "0"
- currency: deteksi mata uang dari simbol (Rp, $, €, ¥), kode (IDR, USD), alamat/lokasi toko, atau istilah pajak (PPN/Pajak)
  - code: ISO 4217 (contoh "IDR"); jika tidak yakin ""
  - symbol: contoh "Rp", "$"; jika tidak ada ""
  - name: nama lengkap (contoh "Indonesian Rupiah"); jika tidak yakin ""
  - confidence: "high" | "medium" | "low"
  - JANGAN ubah format atau isi field lain; currency hanya di object currency
- language: deteksi bahasa utama teks pada struk (label item, header, footer, pajak, total)
  - code: ISO 639-1 (contoh "id", "en", "ja"); jika tidak yakin ""
  - name: nama bahasa dalam English (contoh "Indonesian", "English", "Japanese"); jika tidak yakin ""
  - confidence: "high" | "medium" | "low"
  - jika struk bilingual, pilih bahasa yang paling dominan pada label/teks non-angka
  - JANGAN ubah format atau isi field lain; language hanya di object language

Schema JSON wajib:
{
  "items": [{"name":"","price":"","quantity":"","total":""}],
  "store_information": {"address":"","email":"","npwp":"","phone_number":"","store_name":""},
  "totals": {
    "change":"","discount":"","payment":"","subtotal":"",
    "tax":{"amount":"","service_charge":"","dpp":"","name":"","total_tax":""},
    "total":""
  },
  "transaction_information": {"date":"","time":"","transaction_id":""},
  "currency": {"code":"","symbol":"","name":"","confidence":""},
  "language": {"code":"","name":"","confidence":""}
}`
