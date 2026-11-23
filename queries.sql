WITH bulan AS (
    SELECT date_trunc('month', NOW() AT TIME ZONE 'UTC')                     AS awal,
           date_trunc('month', NOW() AT TIME ZONE 'UTC') + INTERVAL '1 month' AS akhir
),
pemakaian AS (
    SELECT b.room_id,
           SUM(EXTRACT(EPOCH FROM LEAST(b.end_time, bl.akhir) - GREATEST(b.start_time, bl.awal))) AS detik
    FROM bookings b
    CROSS JOIN bulan bl
    WHERE b.status = 'CONFIRMED'
      AND b.start_time < bl.akhir
      AND b.end_time > bl.awal
    GROUP BY b.room_id
)
SELECT r.id,
       r.name,
       ROUND(COALESCE(p.detik, 0) / 3600.0, 2) AS total_jam
FROM rooms r
LEFT JOIN pemakaian p ON p.room_id = r.id
ORDER BY total_jam DESC, r.id;

\set mulai   '2026-12-01 09:00:00'
\set selesai '2026-12-01 11:00:00'

SELECT r.id, r.name, r.capacity
FROM rooms r
WHERE r.is_active
  AND NOT EXISTS (
      SELECT 1
      FROM bookings b
      WHERE b.room_id = r.id
        AND b.status = 'CONFIRMED'
        AND b.start_time < :'selesai'
        AND b.end_time > :'mulai'
  )
ORDER BY r.id;
