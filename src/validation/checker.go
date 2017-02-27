package validation

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

// From https://github.com/asaskevich/govalidator/blob/master/patterns.go
const (
	Email      string = "^(((([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|((\\x22)((((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(([\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(\\([\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(\\x22)))@((([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|\\.|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|\\.|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$"
	CreditCard string = "^(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\\d{3})\\d{11})$"

	Alpha          string = "^[a-zA-Z]+$"
	Alphanumeric   string = "^[a-zA-Z0-9]+$"
	Numeric        string = "^[-+]?[0-9]+$"
	Int            string = "^(?:[-+]?(?:0|[1-9][0-9]*))$"
	Float          string = "^(?:[-+]?(?:[0-9]+))?(?:\\.[0-9]*)?(?:[eE][\\+\\-]?(?:[0-9]+))?$"
	Hexadecimal    string = "^[0-9a-fA-F]+$"
	Hexcolor       string = "^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$"
	RGBcolor       string = "^rgb\\(\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*\\)$"
	ASCII          string = "^[\x00-\x7F]+$"
	Multibyte      string = "[^\x00-\x7F]"
	FullWidth      string = "[^\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]"
	HalfWidth      string = "[\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]"
	Base64         string = "^(?:[A-Za-z0-9+\\/]{4})*(?:[A-Za-z0-9+\\/]{2}==|[A-Za-z0-9+\\/]{3}=|[A-Za-z0-9+\\/]{4})$"
	PrintableASCII string = "^[\x20-\x7E]+$"
	DataURI        string = "^data:.+\\/(.+);base64$"

	DNSName string = `^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9]{1}[a-zA-Z0-9_-]{1,62})*$`
	IP      string = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`

	URLSchema    string = `((ftp|tcp|udp|wss?|https?):\/\/)`
	URLUsername  string = `(\S+(:\S*)?@)`
	Hostname     string = ``
	URLPath      string = `((\/|\?|#)[^\s]*)`
	URLPort      string = `(:(\d{1,5}))`
	URLIP        string = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	URLSubdomain string = `((www\.)|([a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*))`
	URL          string = `^` + URLSchema + `?` + URLUsername + `?` + `((` + URLIP + `|(\[` + IP + `\])|(([a-zA-Z0-9]([a-zA-Z0-9-]+)?[a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*)|(` + URLSubdomain + `?))?(([a-zA-Z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-zA-Z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-zA-Z\x{00a1}-\x{ffff}]{1,}))?))` + URLPort + `?` + URLPath + `?$`

	maxURLRuneCount = 2083
	minURLRuneCount = 3
)

var (
	rxEmail          = regexp.MustCompile(Email)
	rxCreditCard     = regexp.MustCompile(CreditCard)
	rxAlpha          = regexp.MustCompile(Alpha)
	rxAlphanumeric   = regexp.MustCompile(Alphanumeric)
	rxNumeric        = regexp.MustCompile(Numeric)
	rxInt            = regexp.MustCompile(Int)
	rxFloat          = regexp.MustCompile(Float)
	rxHexadecimal    = regexp.MustCompile(Hexadecimal)
	rxHexcolor       = regexp.MustCompile(Hexcolor)
	rxRGBcolor       = regexp.MustCompile(RGBcolor)
	rxASCII          = regexp.MustCompile(ASCII)
	rxPrintableASCII = regexp.MustCompile(PrintableASCII)
	rxBase64         = regexp.MustCompile(Base64)
	rxDataURI        = regexp.MustCompile(DataURI)
	rxDNSName        = regexp.MustCompile(DNSName)
	rxURL            = regexp.MustCompile(URL)
)

type EmailChecker struct{}

func (eck *EmailChecker) Validater(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return NewErrWrongType("string", v)
	}

	if rxEmail.MatchString(str) {
		return nil
	}

	return ErrBadEmailFormat
}

type RequiredChecker struct{}

func (rck *RequiredChecker) Validater(v interface{}) error {
	// Ignore performance, tmp for now
	// ex: beggo: https://github.com/astaxie/beego/blob/master/validation/validators.go#L95
	// Debugf("RequiredChecker %#v", v)
	if reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface()) {
		return ErrRequired
	}

	// Debugf("RequiredChecker passed.")
	return nil
}

type UrlChecker struct{}

func (uck *UrlChecker) Validater(v interface{}) error {

	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("expect string, but got %v", v)
	}

	Debugf("UrlChecker: got [%s]", str)

	if str == "" || utf8.RuneCountInString(str) >= maxURLRuneCount || len(str) <= minURLRuneCount || strings.HasPrefix(str, ".") {
		return ErrBadUrlFormat
	}
	u, err := url.Parse(str)
	if err != nil {
		return ErrBadUrlFormat
	}
	if strings.HasPrefix(u.Host, ".") {
		return ErrBadUrlFormat
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return ErrBadUrlFormat
	}

	if !rxURL.MatchString(str) {
		return ErrBadUrlFormat
	}

	Debugf("UrlChecker: [%s] passed", str)
	return nil
}
