-- Hapus trip yang masih DRAFT (tidak di-save user) dan sudah lebih dari 24 jam
DELETE FROM trips
WHERE (status = 'DRAFT' OR user_id IS NULL)
  AND created_at < NOW() - INTERVAL '24 hours';