package bcrypt

import "golang.org/x/crypto/bcrypt"

type BcryptHasher struct{ Cost int }

func (h BcryptHasher) Hash(plain string) (string, error) {
	cost := h.Cost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	return string(b), err
}

func (BcryptHasher) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
