-- 1. Hapus Constraint Foreign Key terlebih dahulu (agar tidak ada error tipe data)
-- Nama constraint biasanya 'trips_user_id_fkey', tapi kita gunakan command ini jika namanya beda:
ALTER TABLE trips DROP CONSTRAINT IF EXISTS trips_user_id_fkey;

-- 2. Sekarang aman untuk mengubah tipe kolom menjadi VARCHAR
ALTER TABLE trips ALTER COLUMN user_id TYPE VARCHAR(255);

-- 3. (Opsional) Jika kamu mau menghapus tabel users karena pakai Clerk
-- DROP TABLE IF EXISTS users;