package mailer

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"

	"qtp/internal/locale"
)

const brandName = "telebeland"

//go:embed verification_email.html
var verificationHTML string

var verificationTemplate = template.Must(template.New("verification-email").Parse(verificationHTML))

// verificationCopy is the text of one language. Strings never contain user
// data; the link is inserted by html/template, which escapes it.
type verificationCopy struct {
	Subject   string
	Preheader string
	Heading   string
	Greeting  string
	Body      string
	Button    string
	Validity  func(hours int) string
	Fallback  string
	Ignore    string
	Tagline   string
}

var verificationCopies = map[string]verificationCopy{
	locale.Russian: {
		Subject:   "Подтвердите email — telebeland",
		Preheader: "Остался один шаг: подтвердите адрес, чтобы начать подготовку.",
		Heading:   "Подтвердите email",
		Greeting:  "Здравствуйте!",
		Body:      "Спасибо за регистрацию в telebeland. Нажмите кнопку ниже, чтобы подтвердить адрес и войти в учебное пространство.",
		Button:    "Подтвердить email",
		Validity: func(hours int) string {
			return fmt.Sprintf("Ссылка действует %d %s и срабатывает один раз.", hours, russianHours(hours))
		},
		Fallback: "Если кнопка не открывается, скопируйте ссылку в адресную строку браузера:",
		Ignore:   "Если вы не регистрировались в telebeland, просто проигнорируйте это письмо: аккаунт не будет активирован.",
		Tagline:  "Учебное пространство для подготовки к экзаменам",
	},
	locale.Azerbaijani: {
		Subject:   "E-poçtunuzu təsdiqləyin — telebeland",
		Preheader: "Son bir addım qaldı: hazırlığa başlamaq üçün ünvanınızı təsdiqləyin.",
		Heading:   "E-poçtunuzu təsdiqləyin",
		Greeting:  "Salam!",
		Body:      "telebeland-də qeydiyyatdan keçdiyiniz üçün təşəkkür edirik. Ünvanınızı təsdiqləmək və tədris məkanına daxil olmaq üçün aşağıdakı düyməyə klikləyin.",
		Button:    "E-poçtu təsdiqlə",
		Validity: func(hours int) string {
			return fmt.Sprintf("Keçid %d saat ərzində etibarlıdır və yalnız bir dəfə işləyir.", hours)
		},
		Fallback: "Düymə açılmırsa, bu keçidi kopyalayıb brauzerin ünvan sətrinə yapışdırın:",
		Ignore:   "Əgər telebeland-də qeydiyyatdan keçməmisinizsə, bu məktuba əhəmiyyət verməyin: hesab aktivləşdirilməyəcək.",
		Tagline:  "İmtahanlara hazırlıq üçün tədris məkanı",
	},
}

type verificationEmail struct {
	Lang    string
	Subject string
	Plain   string
	HTML    string
}

// renderVerification builds both MIME bodies. The validity is stated in hours
// rather than as a clock time, so it reads correctly in any time zone.
func renderVerification(link string, expires, now time.Time, lang string) (verificationEmail, error) {
	lang = locale.Normalize(lang)
	copy := verificationCopies[lang]
	hours := int(math.Round(expires.Sub(now).Hours()))
	if hours < 1 {
		hours = 1
	}
	validity := copy.Validity(hours)

	var html bytes.Buffer
	if err := verificationTemplate.Execute(&html, struct {
		verificationCopy
		Lang, Link, ValidityText, Brand string
	}{copy, lang, link, validity, brandName}); err != nil {
		return verificationEmail{}, err
	}

	plain := strings.Join([]string{
		copy.Heading,
		"",
		copy.Greeting,
		copy.Body,
		"",
		copy.Button + ":",
		link,
		"",
		validity,
		copy.Ignore,
		"",
		"— " + brandName + " · " + copy.Tagline,
	}, "\r\n") + "\r\n"

	return verificationEmail{Lang: lang, Subject: copy.Subject, Plain: plain, HTML: html.String()}, nil
}

func russianHours(n int) string {
	switch {
	case n%10 == 1 && n%100 != 11:
		return "час"
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		return "часа"
	default:
		return "часов"
	}
}
