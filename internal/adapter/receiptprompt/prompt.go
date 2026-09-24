package receiptprompt

// Extract is the shared OCR instruction for all vision providers.
const Extract = `Lakukan Optical Character Recognition (OCR) pada gambar struk ini dan ekstrak informasi belanja.
Kembalikan HANYA JSON valid sesuai schema, tanpa markdown, tanpa penjelasan, tanpa teks di luar JSON.

Aturan field:
- items: daftar barang/produk dari struk (bukan baris pajak/fee)
- price: harga per unit; jika tidak ada kolom terpisah, hitung total/quantity; jika tidak bisa, "0.00"
- quantity: string angka (contoh "2"); total/price: desimal
- nilai numerik uang: desimal PLAIN tanpa pemisah ribuan (contoh "220000.00"); JANGAN "18.000" atau "18.000,00"; JANGAN sisipkan simbol mata uang
- field tidak ditemukan: null (bukan string kosong "")
- date: DD/MM/YYYY
- time: HH:MM atau null
- discount: angka desimal; jika tidak ada, "0.00"
- store_name: nama toko LENGKAP dari header struk; jangan potong/truncate
- baca nama item hati-hati (menu Indonesia); hindari typo OCR umum

fees[] (WAJIB — source of truth biaya non-item):
- Setiap baris biaya di struk yang BUKAN item produk → 1 entry di fees (pajak, service, tip, packing, delivery, takeaway, round-up, dll)
- Boleh 0 hingga ~10 entry; JANGAN gabungkan beberapa pajak jadi 1 amount kecuali struk hanya menampilkan 1 angka pajak total
- JANGAN truncate / buang baris fee
- type (wajib salah satu): tax | service_charge | tip | fee | other
  Mapping nama → type (case-insensitive):
  - mengandung PB1, PPN, PPNBM, VAT, tax, pajak → tax
  - service, service charge, SC → service_charge
  - tip, gratuity, tips → tip
  - packing, delivery, ongkir, takeaway, kemasan, round → fee
  - tidak yakin → other (tetap masukkan)
- name: teks ASLI label dari struk (pertahankan)
- amount: plain decimal "18564.00"
- rate: jika struk tulis "10%" / "11%" simpan "10" / "11"; else null
- Pajak restoran Indonesia: label PB1/Pajak Daerah — JANGAN tulis "PBB"

Legacy fields (tetap isi agar FE lama tidak pecah; BE akan menghitung ulang dari fees[]):
- totals.tax.amount / total_tax = jumlah semua fee type=tax
- totals.tax.name = nama pajak utama (fee tax terbesar, atau pertama)
- totals.service_charge DAN totals.tax.service_charge = jumlah semua type=service_charge
- totals.tax.dpp: dari struk atau null

Balance:
- sum(items[].total) ≈ totals.subtotal
- totals.subtotal - totals.discount + sum(fees[].amount) ≈ totals.total
- payment: nominal bayar dari struk; jika tidak ada, samakan dengan total
- change: kembalian atau null

currency: deteksi dari simbol (Rp, $, €, ¥), kode (IDR, USD), lokasi, atau istilah pajak
  - code ISO 4217; symbol; name; confidence "high"|"medium"|"low"
language: bahasa utama teks struk
  - code ISO 639-1; name English; confidence "high"|"medium"|"low"
  - bilingual: pilih yang paling dominan pada label non-angka

Schema JSON wajib:
{
  "items": [{"name":"","price":"","quantity":"","total":""}],
  "store_information": {"address":"","email":null,"npwp":null,"phone_number":null,"store_name":""},
  "totals": {
    "subtotal":"",
    "discount":"0.00",
    "fees": [{"type":"tax","name":"","amount":"","rate":null}],
    "tax":{"amount":"","service_charge":"","dpp":null,"name":"","total_tax":""},
    "service_charge":"",
    "total":"",
    "payment":"",
    "change":null
  },
  "transaction_information": {"date":"","time":null,"transaction_id":null},
  "currency": {"code":"","symbol":"","name":"","confidence":""},
  "language": {"code":"","name":"","confidence":""}
}`
