package smartbear_test

import (
	"encoding/json"
	"testing"

	"github.com/purposeinplay/go-commons/smartbear"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestA(t *testing.T) {
	req := require.New(t)

	b, err := json.MarshalIndent(smartbear.ProblemDetails{
		Code:   ptr.To("NOT_ENOUGH_FUNDS"),
		Detail: ptr.To("Not enough balance"),
		Status: ptr.To(int32(400)),
		Title:  "Not enough funds",
	}, " ", "  ")
	req.NoError(err)

	t.Log(string(b))

	b, err = json.MarshalIndent(smartbear.ProblemDetails{
		Status: ptr.To(int32(400)),
		Title:  "Invalid request",
		Errors: ptr.To(smartbear.Errors{
			{
				Code:   ptr.To("NOT_ENOUGH_FUNDS"),
				Detail: "Not enough funds",
			},
		}),
	}, " ", "  ")
	req.NoError(err)

	t.Log(string(b))
}
