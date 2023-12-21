package mau

import (
	"encoding/hex"
	"testing"
)

func TestFingerprintFromString(t *testing.T) {
	t.Run("When converting from and back to string", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6"
		fpr, _ := FingerprintFromString(fprStr)
		fprStrRet := fpr.String()
		ASSERT_EQUAL(t, fprStr, fprStrRet)
	})

	t.Run("With correct fingerprint", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6"
		fpr, err := FingerprintFromString(fprStr)
		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, Fingerprint{118, 229, 166, 16, 205, 203, 216, 74, 168, 28, 248, 51, 23, 19, 223, 227, 22, 54, 129, 214}, fpr)
	})

	t.Run("With short fingerprint", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3"
		fpr, err := FingerprintFromString(fprStr)
		ASSERT_ERROR(t, ErrIncorrectFingerprintLength, err)
		ASSERT_EQUAL(t, Fingerprint{}, fpr)
	})

	t.Run("With long fingerprint", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6777"
		fpr, err := FingerprintFromString(fprStr)
		ASSERT_ERROR(t, ErrIncorrectFingerprintLength, err)
		ASSERT_EQUAL(t, Fingerprint{}, fpr)
	})

	t.Run("With incorrect value fingerprint", func(t T) {
		fprStr := "76g5a610cdcbd84aa81cf8331713dfe3163681d6" // it has "g"
		fpr, err := FingerprintFromString(fprStr)
		ASSERT_EQUAL(t, hex.InvalidByteError('g').Error(), err.Error())
		ASSERT_EQUAL(t, Fingerprint{}, fpr)
	})
}

func TestFingerprint(t *testing.T) {
	t.Run("MarshalJSON", func(t T) {
		fprStr := "76e5a610cdcbd84aa81cf8331713dfe3163681d6"
		fpr, _ := FingerprintFromString(fprStr)
		jsonStr, err := fpr.MarshalJSON()
		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, `"`+fprStr+`"`, string(jsonStr))
	})

	t.Run("UnmarshalJSON", func(t T) {
		t.Run("With correct string", func(t T) {
			fprStr := `"76e5a610cdcbd84aa81cf8331713dfe3163681d6"`
			var fpr Fingerprint
			err := fpr.UnmarshalJSON([]byte(fprStr))
			ASSERT_NO_ERROR(t, err)
			ASSERT_EQUAL(t, Fingerprint{118, 229, 166, 16, 205, 203, 216, 74, 168, 28, 248, 51, 23, 19, 223, 227, 22, 54, 129, 214}, fpr)
		})

		t.Run("With null string", func(t T) {
			fprStr := `null`
			var fpr Fingerprint
			err := fpr.UnmarshalJSON([]byte(fprStr))
			ASSERT_NO_ERROR(t, err)
			ASSERT_EQUAL(t, Fingerprint{}, fpr)
		})
	})
}
