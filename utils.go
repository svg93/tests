package main

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	// descCharacterLimit содержит лимит символов для тега Description,
	// установленный согласно документации Авито.
	// См. https://www.avito.ru/autoload/documentation/templates/67029?fileFormat=xml#field-Description
	descCharacterLimit = 7500

	// trimmedDescCharacterLimit содержит лимит символов
	// для обрезанного тега Description.
	trimmedDescCharacterLimit = 7497
)

var (
	regexpN  = regexp.MustCompile(`(\n{2,})`)
	regexpRN = regexp.MustCompile(`(\r\n{2,})`)
	words    = regexp.MustCompile("[\\p{L}\\d_]+")
)

// encMap - константный массив для функции encode4chars.
var encMap = map[string]string{
	"0": "7", "1": "8", "2": "9", "3": "a", "4": "b", "5": "c", "6": "d", "7": "e",
	"8": "f", "9": "0", "a": "1", "b": "2", "c": "3", "d": "4", "e": "5", "f": "6",
}

func createTmpDir() (string, error) {

	var err error

	name := os.TempDir() + "/" + stageName

	deleteFile(name)

	err = os.Mkdir(name, 0700)
	if os.IsExist(err) {
		log.Warnf("Временный каталог %s не удалялся с предыдущего цикла. ", name)
	}
	if os.IsNotExist(err) {
		if _, err = os.Stat(os.TempDir()); os.IsNotExist(err) {
			return "", err
		}
	}

	name += "/"

	return name, nil
}

func deleteFile(name string) {
	if err := os.RemoveAll(name); err != nil {
		log.Errorf("Не удалось удалить файл (%s), ошибка: %s", name, err)
	}
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any)
	for key, value := range m {
		out[key] = value
	}

	return out
}

// replaceSeparatorToComma заменяет десятичный разделить
// в дробном числе с точки на запятую.
func replaceSeparatorToComma(str string) string {
	return strings.ReplaceAll(str, ".", ",")
}

func caseInsensitiveContains(s, substr string) bool {

	s, substr = strings.ToUpper(s), strings.ToUpper(substr)

	return strings.Contains(s, substr)
}

func buildFinalOfferDescription(descr, salesConditions string) string {

	d := descr + salesConditions

	d = regexpN.ReplaceAllString(d, "\n")
	d = regexpRN.ReplaceAllString(d, "\r\n")
	d = strings.ReplaceAll(d, "\r\n", "\n")
	d = strings.ReplaceAll(d, "\n", "<br/>")

	if len([]rune(d)) > descCharacterLimit {
		return string([]rune(d)[0:trimmedDescCharacterLimit]) + "..."
	}

	return d
}

// replaceModAutorus преобразовывает части формирующегося URL для альтернативного
// отображения изображения согласно документации от autorus.ru.
func replaceModAutorus(positionName string, isNumber bool) string {

	s := strings.ToLower(positionName)
	s = strings.Replace(s, "/", "-", -1)
	if isNumber {
		s = strings.Replace(s, " ", "", -1)
	}
	replacer := strings.NewReplacer(
		"\\", "",
		".", "",
		",", "",
		"\"", "",
		"\\'", "",
		"\r", "",
		"\n", "",
		"\r\n", "",
	)

	return replacer.Replace(s)
}

func getString(i any) string {

	if i == nil {
		return ""
	}

	switch val := i.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', 2, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', 2, 64)
	case []byte:
		return string(val)
	default:
		return ""
	}
}

// encode4chars кодирует имя картинки по ТЗ от аналитики.
//
// ТЗ в виде комментариев в telegram-чате от PVA (заявки нет).
// =======
// Алгоритм модификации имени картинки
//
// Нормальное имя: 05375f5bf51cd0207d02e5655f14e89cb.jpeg
// 34 символа в имени без точки и расширения
//
// Модифицированное имя: 05375f_5bf51c_d0207d_02e565_5f14e89cb.jpeg
// 38 символов в имени без точки и расширения
// Вместо символа "_" будет рандомный символ из набора 0-9a-f
// Позиции в которые ВСТАВЛЯЮТСЯ символы: 7,14,21,28
//
// Вместо символа "_" будет сдвинутый на +7 символ из позиций 24,32,6,12
// Позиции в которые ВСТАВЛЯЮТСЯ символы: 7,14,21,28
// Сдвиг +7, это значит 0=7, 1=8, 2=9, 3=a, 4=b ... f=6
//
// Т.е. берутся символы из позиций 24,32,6,12
// сдвигаются и вставляются на позиции 7,14,21,28
func encode4chars(s string) string {

	if len(s) < 34 {
		return s
	}

	return s[0:6] + encMap[s[24:25]] +
		s[6:12] + encMap[s[32:33]] +
		s[12:18] + encMap[s[6:7]] +
		s[18:24] + encMap[s[12:13]] +
		s[24:]
}

func getWordsFrom(text string) []string {
	return words.FindAllString(text, -1)
}

// removeCatTagsHyphen удаляет дефис из значений тегов категории,
// если кроме них в строке больше нет символов.
func removeCatTagsHyphen(catTags AvitoCategoriesTagsStruct, table string) AvitoCategoriesTagsStruct {

	if table == "TruckDescrCategories" {
		if strings.TrimSpace(catTags.TechnicSparePartType) == "-" {
			catTags.TechnicSparePartType = ""
		}
	}

	if table == "DescrCategories" || table == "BrandCategories" || table == "Categories" {
		if strings.TrimSpace(catTags.ProductType) == "-" {
			catTags.ProductType = ""
		}
	}

	if table == "TruckDescrCategories" || table == "DescrCategories" || table == "BrandCategories" || table == "Categories" {
		if strings.TrimSpace(catTags.SparePartType) == "-" {
			catTags.SparePartType = ""
		}
	}

	if table == "DescrCategories" || table == "BrandCategories" || table == "Categories" {
		if strings.TrimSpace(catTags.GoodsType) == "-" {
			catTags.GoodsType = ""
		}
	}

	if table == "DescrCategories" || table == "BrandCategories" || table == "Categories" {
		if strings.TrimSpace(catTags.Category) == "-" {
			catTags.Category = ""
		}
	}

	if table == "DescrCategories" || table == "Categories" {
		if strings.TrimSpace(catTags.SparePartType2) == "-" {
			catTags.SparePartType2 = ""
		}
	}

	return catTags
}
