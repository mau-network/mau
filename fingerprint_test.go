package mau

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFingerprintFromString(t *testing.T) {
	t.Run("When converting from and back to string", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6"
		fpr, _ := FingerprintFromString(fprStr)
		fprStrRet := fpr.String()
		assert.Equal(t, fprStr, fprStrRet)
	})

	t.Run("With correct fingerprint", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6"
		fpr, err := FingerprintFromString(fprStr)
		assert.NoError(t, err)
		assert.Equal(t, Fingerprint{118, 229, 166, 16, 205, 203, 216, 74, 168, 28, 248, 51, 23, 19, 223, 227, 22, 54, 129, 214}, fpr)
	})

	t.Run("With short fingerprint", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3"
		fpr, err := FingerprintFromString(fprStr)
		assert.Error(t, ErrIncorrectFingerprintLength, err)
		assert.Equal(t, Fingerprint{}, fpr)
	})

	t.Run("With long fingerprint", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6777"
		fpr, err := FingerprintFromString(fprStr)
		assert.Error(t, ErrIncorrectFingerprintLength, err)
		assert.Equal(t, Fingerprint{}, fpr)
	})

	t.Run("With incorrect value fingerprint", func(t T) {
		fprStr := "76g5a610cdcbd84aa81cf8331713dfe3163681d6" // it has "g"
		fpr, err := FingerprintFromString(fprStr)
		assert.Equal(t, hex.InvalidByteError('g').Error(), err.Error())
		assert.Equal(t, Fingerprint{}, fpr)
	})
}

func TestFingerprint(t *testing.T) {
	t.Run("MarshalJSON", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6"
		fpr, _ := FingerprintFromString(fprStr)
		jsonStr, err := fpr.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, `"`+fprStr+`"`, string(jsonStr))
	})

	t.Run("UnmarshalJSON", func(t T) {
		t.Run("With correct string", func(t T) {
			fprStr := `"76e5a610cdcbd84aa81cf8331713dfe3163681d6"`
			var fpr Fingerprint
			err := fpr.UnmarshalJSON([]byte(fprStr))
			assert.NoError(t, err)
			assert.Equal(t, Fingerprint{118, 229, 166, 16, 205, 203, 216, 74, 168, 28, 248, 51, 23, 19, 223, 227, 22, 54, 129, 214}, fpr)
		})

		t.Run("With null string", func(t T) {
			fprStr := `null`
			var fpr Fingerprint
			err := fpr.UnmarshalJSON([]byte(fprStr))
			assert.NoError(t, err)
			assert.Equal(t, Fingerprint{}, fpr)
		})
	})
}
