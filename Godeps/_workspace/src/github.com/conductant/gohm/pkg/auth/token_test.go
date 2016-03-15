package auth

import (
	"github.com/conductant/gohm/pkg/testutil"
	. "gopkg.in/check.v1"
	"testing"
	"time"
)

func TestToken(t *testing.T) { TestingT(t) }

type TestSuiteToken struct {
}

var _ = Suite(&TestSuiteToken{})

func (suite *TestSuiteToken) SetUpSuite(c *C) {
}

func (suite *TestSuiteToken) TearDownSuite(c *C) {
}

func (suite *TestSuiteToken) TestParseKeys(c *C) {
	_, err := RsaPublicKeyFromPem(testutil.PublicKeyFunc)
	c.Assert(err, IsNil)
	_, err = RsaPrivateKeyFromPem(testutil.PrivateKeyFunc)
	c.Assert(err, IsNil)
}

func (suite *TestSuiteToken) TestNewToken(c *C) {

	token := NewToken(1 * time.Hour)
	token.Add("foo1", "foo1").Add("foo2", "foo2").Add("count", 2)

	signedString, err := token.SignedString(testutil.PrivateKeyFunc)
	c.Assert(err, IsNil)

	c.Log("token=", signedString)
	parsed, err := TokenFromString(signedString, testutil.PublicKeyFunc, time.Now)
	c.Assert(err, IsNil)
	c.Assert(parsed.HasKey("count"), Equals, true)
	c.Assert(parsed.Get("count"), Equals, float64(2))
	c.Assert(parsed.Get("foo1"), DeepEquals, "foo1")
}

func (suite *TestSuiteToken) TestAuthTokenExpiration(c *C) {

	token := NewToken(1 * time.Hour)
	encoded, err := token.SignedString(testutil.PrivateKeyFunc)

	// decode
	_, err = TokenFromString(encoded, testutil.PublicKeyFunc, func() time.Time { return time.Now().Add(2 * time.Hour) })
	c.Assert(err, Equals, ErrExpiredAuthToken)
}

type uuid string

func (suite *TestSuiteToken) TestGetAppAuthTokenAuthRsaKey(c *C) {

	id := uuid("1234")

	token := NewToken(1*time.Hour).Add("appKey", id)
	encoded, err := token.SignedString(testutil.PrivateKeyFunc)

	// decode
	parsed, err := TokenFromString(encoded, testutil.PublicKeyFunc, time.Now)
	c.Assert(err, IsNil)

	appKey := parsed.GetString("appKey")
	c.Log("appkey=", appKey)
	c.Assert(uuid(appKey), DeepEquals, id)
}
