package main

import (
	"bufio"
	"embed"
	_ "embed"
	"log"
	"sort"
	"strings"
)

var localeMap = make(map[string]string)

//go:embed i18n/english.txt
var english []byte

//go:embed i18n
var i18nDir embed.FS

var local_SMap = map[string]string{
	"English":                                           "english",
	"简体中文 (Simplified Chinese)":                         "schinese",
	"繁體中文 (Traditional Chinese)":                        "tchinese",
	"日本語 (Japanese)":                                    "japanese",
	"한국어 (Korean)":                                      "koreana",
	"ไทย (Thai)":                                        "thai",
	"Български (Bulgarian)":                             "bulgarian",
	"Čeština (Czech)":                                   "czech",
	"Dansk (Danish)":                                    "danish",
	"Deutsch (German)":                                  "german",
	"Español - España (Spanish - Spain)":                "spanish",
	"Español - Latinoamérica (Spanish - Latin America)": "latam",
	"Ελληνικά (Greek)":                                  "greek",
	"Français (French)":                                 "french",
	"Italiano (Italian)":                                "italian",
	"Bahasa Indonesia (Indonesian)":                     "indonesian",
	"Magyar (Hungarian)":                                "hungarian",
	"Nederlands (Dutch)":                                "dutch",
	"Norsk (Norwegian)":                                 "norwegian",
	"Polski (Polish)":                                   "polish",
	"Português (Portugal)":                              "portuguese",
	"Português - Brasil (Portuguese - Brazil)":          "brazilian",
	"Română (Romanian)":                                 "romanian",
	"Русский (Russian)":                                 "russian",
	"Suomi (Finnish)":                                   "finnish",
	"Svenska (Swedish)":                                 "swedish",
	"Türkçe (Turkish)":                                  "turkish",
	"Tiếng Việt (Vietnamese)":                           "vietnamese",
	"Українська (Ukrainian)":                            "ukrainian",
}

func InitI18n(i18n string) string {
	var i18nCode = strings.ToLower(i18n)
	if strings.Contains(i18nCode, "chinese") {
		if strings.Contains(i18nCode, "simplified") || i18nCode == "schinese" {
			i18nCode = "schinese"
		} else {
			i18nCode = "tchinese"
		}
	} else if strings.Contains(i18nCode, "spanish") {
		if strings.Contains(i18nCode, "latin") {
			i18nCode = "latam"
		} else {
			i18nCode = "spanish"
		}
	} else if strings.Contains(i18nCode, "portuguese") {
		if strings.Contains(i18nCode, "Brazil") {
			i18nCode = "brazilian"
		} else {
			i18nCode = "portuguese"
		}
	} else if strings.Contains(i18nCode, "korean") {
		i18nCode = "koreana"
	}

	i18nData, err := i18nDir.ReadFile("i18n/" + i18nCode + ".txt")
	if err != nil {
		i18nData = english
	}

	reader := strings.NewReader(string(i18nData))

	// 创建一个 Scanner 来读取文件
	scanner := bufio.NewScanner(reader)

	// 按行读取文件内容
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, "=")
		if len(split) == 1 {
			localeMap[strings.TrimSpace(split[0])] = strings.TrimSpace(split[0])
		} else {
			localeMap[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
		}
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		log.Println("[ERROR] read i18n: ", i18n)
		return i18nCode
	}

	log.Println("[Info] read i18n: ", i18n, " Success")
	return i18nCode
}

func Text(text string) string {
	targetText, ok := localeMap[text]
	if ok {
		return targetText
	}
	targetText, ok = localeMap[strings.ToLower(text)]
	if ok {
		return targetText
	}
	return text
}

func GetLocaleMap() map[string]string {
	return localeMap
}

func GetLocaleLangs() []string {
	var languages []string
	for k := range local_SMap {
		languages = append(languages, k)
	}
	sort.Strings(languages)
	sort.Slice(languages, func(i, j int) bool {
		a, b := languages[i], languages[j]

		if a == "English" || a == "简体中文 (Simplified Chinese)" || a == "繁體中文 (Traditional Chinese)" {
			return true
		}
		return a < b
	})
	return languages
}

func GetLocaleLangsCode(lang string) string {
	code, isok := local_SMap[lang]
	if isok {
		return code
	}
	return "english"
}

func GetLocaleCodeIndex(code string) int {
	index := 0

	langs := GetLocaleLangs()

	for _, l := range langs {
		val, _ := local_SMap[l]
		if val == code {
			return index
		}
		index++
	}

	return 0
}
