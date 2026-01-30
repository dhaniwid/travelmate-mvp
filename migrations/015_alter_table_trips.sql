-- Ubah tipe data kolom style dari VARCHAR(50) menjadi TEXT (unlimited)
-- atau VARCHAR(255) agar aman menampung deskripsi vibe yang panjang.

ALTER TABLE trips
    ALTER COLUMN style TYPE TEXT;