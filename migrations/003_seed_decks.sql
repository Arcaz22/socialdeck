-- Seed deck default sistem
INSERT INTO decks (id, name, mode, is_public, is_system, created_by)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'Truth or Dare Classic', 'truth_or_dare', true, true, NULL),
    ('00000000-0000-0000-0000-000000000002', 'Truth or Truth', 'truth_or_truth', true, true, NULL),
    ('00000000-0000-0000-0000-000000000003', 'Talk More', 'talk_more', true, true, NULL)
ON CONFLICT (id) DO NOTHING;

-- Cards: Truth or Dare Classic
INSERT INTO cards (deck_id, type, content) VALUES
    ('00000000-0000-0000-0000-000000000001', 'truth', 'Apa hal paling memalukan yang pernah kamu lakukan?'),
    ('00000000-0000-0000-0000-000000000001', 'truth', 'Siapa orang yang paling kamu rindukan saat ini?'),
    ('00000000-0000-0000-0000-000000000001', 'truth', 'Apa rahasia terbesar yang belum pernah kamu ceritakan ke siapapun?'),
    ('00000000-0000-0000-0000-000000000001', 'truth', 'Pernahkah kamu berbohong kepada sahabat terbaikmu? Tentang apa?'),
    ('00000000-0000-0000-0000-000000000001', 'truth', 'Apa hal yang paling kamu sesali dalam hidupmu?'),
    ('00000000-0000-0000-0000-000000000001', 'dare', 'Telepon seseorang secara acak dari kontakmu dan nyanyikan lagu ulang tahun'),
    ('00000000-0000-0000-0000-000000000001', 'dare', 'Lakukan 20 push-up sekarang'),
    ('00000000-0000-0000-0000-000000000001', 'dare', 'Posting foto selfie tanpa filter di story selama 1 jam'),
    ('00000000-0000-0000-0000-000000000001', 'dare', 'Minta nomor telepon orang asing terdekat'),
    ('00000000-0000-0000-0000-000000000001', 'dare', 'Bicara dengan aksen berbeda selama 3 ronde berikutnya')
ON CONFLICT (deck_id, type, content) DO NOTHING;

-- Cards: Truth or Truth
INSERT INTO cards (deck_id, type, content) VALUES
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Apa hal yang paling kamu takuti dalam hubungan?'),
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Kalau bisa mengulang satu momen dalam hidupmu, apa itu?'),
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Apa pencapaian yang paling kamu banggakan tapi jarang kamu ceritakan?'),
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Siapa di ruangan ini yang paling kamu kagumi dan kenapa?'),
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Apa hal yang ingin kamu ubah dari dirimu sendiri?'),
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Kapan terakhir kali kamu menangis dan karena apa?'),
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Apa mimpi yang sudah lama kamu pendam tapi belum pernah kamu kejar?'),
    ('00000000-0000-0000-0000-000000000002', 'truth', 'Hal apa yang sering kamu pura-pura oke padahal tidak?')
ON CONFLICT (deck_id, type, content) DO NOTHING;

-- Cards: Talk More
INSERT INTO cards (deck_id, type, content) VALUES
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Ceritakan momen ketika kamu merasa paling bahagia tahun ini'),
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Kalau kamu bisa makan malam bersama siapapun di dunia, siapa itu?'),
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Apa hal kecil yang bisa membuat harimu jauh lebih baik?'),
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Ceritakan tentang seseorang yang sangat berpengaruh dalam hidupmu'),
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Apa hal yang ingin kamu pelajari tapi belum sempat?'),
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Kalau kamu punya satu hari bebas tanpa tanggung jawab, kamu mau ngapain?'),
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Apa arti persahabatan bagimu?'),
    ('00000000-0000-0000-0000-000000000003', 'truth', 'Ceritakan hal yang membuatmu bersemangat belakangan ini')
ON CONFLICT (deck_id, type, content) DO NOTHING;
