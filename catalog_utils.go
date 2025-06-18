package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	log "github.com/sirupsen/logrus"
)

type currentTime struct {
	time.Time
}

// buildOfferName формирует содержимое тега <Title>.
func buildOfferName(pos position, props map[string]any, localization Localization, priorityDescriptionSource int) string {

	// Заменяем описание позиции описанием сформированным на этапе datacollector для следующих приоритетных источников описания:
	// 1. "Справочник товаров + Прайс-лист + параметры товара" (priorityDescriptionSource = 5)
	// 	  Поле TitleDescription заполняется из справочника Article API или значения по-умолчанию "Деталь".
	// 2. Информация о товарах (Описание и Описание для маркетплейсов) (priorityDescriptionSource = 7)
	//    Поле TitleDescription заполняется из "Описания" карточки товара, из прайс-листа или значения по-умолчанию "Деталь".
	if priorityDescriptionSource == 5 || priorityDescriptionSource == 7 {
		pos.Description = pos.TitleDescription
	}

	var offerName string
	if pos.DescriptionSource == 0 {
		offerName = getOfferNameByCategory(pos, props)
	} else {
		offerName = pos.Description
	}

	offerName = deleteNonBreakingSpace(offerName)

	if pos.Condition != 0 {
		offerName = localization.WearoutPreOwned + " " + offerName
	}

	nameRunes := bytes.Runes([]byte(offerName))
	if len(nameRunes) > 50 {
		offerName = trimOfferNameByWords(offerName)
	}

	return offerName
}

// getOfferNameByCategory формирует название позиции товара в зависимости от его категории.
func getOfferNameByCategory(pos position, props map[string]any) string {

	var offerName string
	switch pos.Category {
	case "tires":
		offerName = "Шины " + pos.Brand + " " +
			getString(props["catalog_model"]) + " " +
			getString(props["width"]) + "/" +
			getString(props["height"]) + "R" +
			getString(props["diameter"]) + " " +
			getString(props["load_index"]) + " " +
			getString(props["speed_index"])
	case "disks":
		offerName = "Диск " + pos.Brand + ", " +
			getString(props["catalog_model"]) + " " +
			getString(props["width"]) + "x" +
			getString(props["diameter"]) + "/" +
			getString(props["holes"]) + "x" +
			getString(props["pcd"]) + "ET" +
			getString(props["et"]) + " " +
			getString(props["hub_diameter"])
	case "oils":
		offerName = pos.Description + " " + pos.Brand
	default:
		var number string
		if pos.CustomNumber != "" {
			number = pos.CustomNumber
		} else {
			number = pos.Number
		}
		offerName = pos.Description + " " + number + " " + pos.Brand
	}

	return offerName
}

// deleteNonBreakingSpace удаляет неразрывный пробел из названия,
// заменяя его стандартным пробелом.
func deleteNonBreakingSpace(name string) string {

	name = strings.ReplaceAll(name, "<0xa0>", " ")
	name = strings.ReplaceAll(name, "\u00a0", " ")
	return strings.ReplaceAll(name, "&nbsp;", " ")
}

// trimOfferNameByWords удаляет слова из наименования позиции.
func trimOfferNameByWords(offerName string) string {

	nameElems := strings.Split(offerName, " ")
	for i := 0; i < len(nameElems); {

		nameElems[i] = strings.TrimSpace(nameElems[i])
		nameElemRunes := bytes.Runes([]byte(nameElems[i]))

		if nameElems[i] == "" {
			nameElems = append(nameElems[:i], nameElems[i+1:]...)
			continue
		} else if len(nameElemRunes) > 50 {
			nameElems[i] = string(nameElemRunes[:50])
		}

		i++
	}

	if len(nameElems) == 1 && len(bytes.Runes([]byte(nameElems[0]))) <= 50 {
		return nameElems[0]
	} else if len(nameElems) == 0 {
		return ""
	}

	nameElems = deleteWords(nameElems)
	nameElems = deleteFuncWords(nameElems)

	return strings.Join(nameElems, " ")
}

// deleteWords удаляет слова из названия позиции,
// пока его длина не станет меньше или равной 50.
func deleteWords(nameElems []string) []string {

	name := strings.Join(nameElems, " ")
	nameRunes := bytes.Runes([]byte(name))

	if len(nameRunes) > 50 {
		nameElems = nameElems[:len(nameElems)-1]
		return deleteWords(nameElems)
	}

	return nameElems
}

// deleteFuncWords удаляет служебные слова из названия позиции,
// пока они будут последними.
func deleteFuncWords(nameElems []string) []string {

	if len(nameElems) == 0 {
		return nameElems
	}

	isFuncWord := false
	funcWords := []string{"и", "в", "к", "c", "с"}
	for _, word := range funcWords {
		if nameElems[len(nameElems)-1] == word {
			isFuncWord = true
			break
		}
	}

	if isFuncWord {
		nameElems = nameElems[:len(nameElems)-1]
		return deleteFuncWords(nameElems)
	}

	return nameElems
}

// getSet возвращает значение тега <Set> для группы товаров wipers.
func getSet(propPackCount any) string {

	switch propPackCount.(string) {
	case "1":
		return "Нет"
	case "2":
		return "Да"
	default:
		return ""
	}
}

// getMountingType возвращает значение тега <MountingType> для группы товаров wipers.
func getMountingType(propConnector any) string {

	mountingTypeMap := map[string]string{
		"j-hook (крючок)":                        "Hook 9x4",
		"bayonet (штыковой замок)":               "Bayonet arm",
		"side pin (боковой штырь) 22мм":          "Side pin 22",
		"push button (кнопка) 19мм":              "Push button 16",
		"narrow push putton (узкая кнопка) 16мм": "Narrow push button",
		"pinch tab (боковой зажим)":              "Pinch tab",
		"top lock (верхний замок)":               "Top lock",
		"claw (клешня)":                          "Claw",
		"pin lock (штырь)":                       "Pin lock",
		"side pin (боковой штырь) 17мм":          "Side pin 17",
		"side mounting (боковое крепление)":      "Side mounting",
		"GWB046 (VATL5.1)":                       "VATL 5.1",
		"special (специальное)":                  "Hook 9x4",
		"штырь 4.8/6.5 мм":                       "Pin lock",
		"GWB044 (DNTL1.1)":                       "DNTL 1.1",
		"GWB045 (MBTL1.1)":                       "MBTL 1.1",
		"DYTL1.1":                                "DYTL 1.1",
		"грузовой крючок 27/6":                   "Hook 9x4",
		"AeroClip (АэроКлип)":                    "AeroClip",
		"GWB071":                                 "Hook 9x4",
		"грузовой крючок 25/6":                   "Hook 9x4",
		"грузовой крючок 22/6":                   "Hook 9x4",
		"RBTL2.0 (19мм)":                         "Hook 9x4",
	}

	switch v := propConnector.(type) {
	case []any:
		return mountingTypeMap[v[0].(string)]
	case string:
		return mountingTypeMap[v]
	default:
		return ""
	}
}

// getMountingType возвращает значение тегов
// <BrushLength> и <SecondBrushLength> для группы товаров wipers.
// Параметр num обозначает номер параметра length из article api.
func getBrushLength(propLen any, num string) int {

	lenStr, ok := propLen.(string)
	if !ok {
		return propLen.(int)
	}

	length, err := strconv.Atoi(lenStr)
	if err != nil {
		log.Errorf("Could not convert prop length%s %q for group wipers: %v\n", num, lenStr, err)
		return 0
	}

	return length
}

func buildAvailability(avail, deadline int) string {

	switch avail {
	case 0:
		if deadline > 0 {
			return "Под заказ"
		}
		return "В наличии"
	case 1:
		return "В наличии"
	case 2:
		return "Под заказ"
	default:
		return ""
	}
}

func getTyreType(season string) string {

	switch season {
	case "всесезонная":
		return "Всесезонные"
	case "зимняя нешипованная":
		return "Зимние нешипованные"
	case "зимняя шипованная":
		return "Зимние шипованные"
	case "летняя":
		return "Летние"
	default:
		return ""
	}
}

func buildOfferDescription(pos position, productDetail string, priorityDescSource int, loc Localization, removeStmtTypeFromDesc bool) string {

	var (
		d             string
		statementType string
		number        string
	)

	if pos.CustomNumber != "" {
		number = pos.CustomNumber
	} else {
		number = pos.Number
	}
	d = "Бренд: " + pos.Brand + ", артикул: " + number + ", "

	if pos.Condition != 0 {

		if removeStmtTypeFromDesc {
			d = d + loc.WearoutPreOwned + " "
		} else {
			switch pos.Condition {
			case 0:
				statementType = loc.StatementTypeNew
			case 10:
				statementType = loc.StatementTypePerfrect
			case 30:
				statementType = loc.StatementTypeGood
			case 50:
				statementType = loc.StatementTypeNormal
			case 70:
				statementType = loc.StatementTypeBroken
			case 90:
				statementType = loc.StatementTypeRepairKit
			}
			d = d + loc.WearoutPreOwned + ". " + loc.ListThBuStatement + " - " + statementType + ", "
		}
	}

	switch priorityDescSource {
	case 2:
		if pos.AdditionalDescription == "" {
			d = d + pos.Description + ". <br/>"
		} else {
			d = d + pos.AdditionalDescription + ". <br/>"
		}
	case 3:
		if pos.AdditionalDescription == "" {
			d = d + pos.Description
		} else {
			d = d + pos.AdditionalDescription
		}
		d = d + ". <br/>" + productDetail
	case 4:
		if pos.AdditionalDescription != "" {
			d = d + pos.AdditionalDescription + ". <br/>"
		}
		d = d + productDetail
	case 7:
		d = d + pos.Description + ". <br/>"
	default:
		d = d + pos.Description + ". <br/>" + productDetail
	}

	return d
}

// getOfferIDWhCode формирует уникальный идентификатор предложения из внутреннего кода
// или, при отсутствии кода, из брендa/артикула.
func getOfferIDWhCode(pos position) string {

	whCode := strings.TrimSpace(pos.Code)
	if len(whCode) > 0 {
		return whCode
	}

	b := strings.ReplaceAll(pos.Brand, "ё", "е")
	b = strings.ReplaceAll(b, "&", "_")
	b = strings.ReplaceAll(b, " ", "")

	return b + "-" + pos.Number
}

func buildOfferID(pos position, avitoOfferID int) string {
	switch avitoOfferID {
	case 0:
		data := []byte(pos.Number + pos.Brand + strconv.Itoa(pos.RouteID))

		s := fmt.Sprintf("%x", md5.Sum(data))

		return s[0:20]
	case 1:
		data := []byte(pos.Number + pos.Brand)

		s := fmt.Sprintf("%x", md5.Sum(data))

		return s[0:20]
	case 2:
		return pos.Brand + "_" + pos.Number
	case 3:
		return getOfferIDWhCode(pos)
	}

	return ""
}

// buildDateEnd формирует значение тега <DateEnd>.
func buildDateEnd(curTime time.Time, endDate int) string {

	switch endDate {
	case 1:
		return curTime.Add(time.Hour * 24).Format(time.DateOnly)
	case 2:
		return curTime.Add(time.Hour * 24 * 2).Format(time.DateOnly)
	case 3:
		return curTime.Add(time.Hour * 24 * 3).Format(time.DateOnly)
	case 4:
		return curTime.Add(time.Hour * 24 * 4).Format(time.DateOnly)
	case 5:
		return curTime.Add(time.Hour * 24 * 5).Format(time.DateOnly)
	case 6:
		return curTime.Add(time.Hour * 24 * 6).Format(time.DateOnly)
	case 7:
		return curTime.Add(time.Hour * 24 * 7).Format(time.DateOnly)
	case 8:
		return curTime.Add(time.Hour * 24 * 10).Format(time.DateOnly)
	case 9:
		return curTime.Add(time.Hour * 24 * 14).Format(time.DateOnly)
	case 10:
		return curTime.Add(time.Hour * 24 * -1).Format(time.DateOnly)
	default:
		return ""
	}
}

// addWipersParams добавляет значения параметров
// "Место установки" и "Производитель" в []xmlParam
// только для группы товаров wipers (щётки стеклоочистителя),
// для которых нет данных в позициях прайса.
func addWipersParams(brand string) []xmlParam {
	return []xmlParam{
		{Name: "Место установки", Content: "Лобовое стекло"},
		{Name: "Производитель", Content: brand},
	}
}

func buildOfferModel(inputModel string, avitoModels AvitoModelsStruct, regexpTiresModel *regexp.Regexp) string {

	if inputModel == "" {
		return ""
	}

	fixedModel, err := removeRedundantCharactersTireModels(inputModel, regexpTiresModel)
	if err != nil {
		log.Errorf("Ошибка приведения формата моделей авито")
		return ""
	}
	for _, model := range avitoModels.Response {
		if model.Manual {
			if strings.ToUpper(model.AbcpName) == fixedModel {
				return model.AvitoName
			}
		}

		if fixedModel == model.AvitoNameFix {
			return model.AvitoName
		}
	}
	return inputModel
}

// removeRedundantCharacters удаляет лишние символы из строки.
func removeRedundantCharactersTireModels(tireModelName string, regexpTiresModel *regexp.Regexp) (string, error) {

	str := regexpTiresModel.ReplaceAllString(tireModelName, "")
	out := strings.ToUpper(str)

	return out, nil
}

func buildProductDetail(xmlParams []xmlParam) string {

	var (
		li  []string
		res string
	)

	for _, v := range xmlParams {
		li = append(li, "<li>"+v.Name+": "+v.Content+".</li>")
	}

	if len(li) > 0 {
		res = "<ul>" + strings.Join(li, " ") + "</ul>" + " "
	}

	return res
}

// getApplicabilityProp получает значение свойства applicability
// для категории товаров gear_oils.
func getApplicabilityProp(props map[string]any) string {

	if appProp, ok := props["applicability"]; ok {
		switch prop := appProp.(type) {
		case []any:
			return prop[0].(string)
		default:
			return ""
		}
	}

	return ""
}

// buildVehicleType устанавливает значение типа транспортного средства для выходного xml-файла.
func buildVehicleType(props map[string]any) string {

	var vehicleType string
	if props["age"] == "для детей" || props["age"] == "для подростков" {
		vehicleType = "Детские"
	} else {
		if props["type"] == "горный" {
			vehicleType = "Горные"
		}
		if props["type"] == "bmx" {
			vehicleType = "BMX"
		}
	}

	if vehicleType == "" {
		vehicleType = "Дорожные"
	}

	return vehicleType
}

// getAbcpATFByType получает значение abcp параметра atf_spec по типу данных.
func getAbcpATFByType(prop any) string {

	switch val := prop.(type) {
	case string:
		return val
	case []any:
		return val[0].(string)
	default:
		return ""
	}
}

// getATF формирует значение тега <ATF>.
// Если есть соответствующее значение из Avito, возвращаем его,
// иначе abcp параметр atf_spec.
func getATF(avitoAtfSpec map[string]string, abcpATF string) string {

	if avitoATF, ok := avitoAtfSpec[abcpATF]; ok {
		return avitoATF
	}

	return abcpATF
}

func newCharData(s string) charData {
	return charData{[]byte("<![CDATA[" + s + "]]>")}
}

// addImageIncParam добавляет в url-адрес картинки параметр "?<ImageInc>", если он больше нуля.
func addImageIncParam(imgURL string, imageInc int) string {

	if imageInc > 0 {
		imgURL += "?" + strconv.Itoa(imageInc)
	}

	return imgURL
}

// buildImgURLOffer возвращает ссылку на изображение для:
// стандартного отображения - https://pubimg.4mycar.ru/images/
// альтернативного отображения - https://img.autorus.ru/
func buildImgURLOffer(imgName, pfx, alternativeImageProxy, brand, number string, disableAlternativeProxy bool, updatePhoto bool, updatePhotoCount int) string {

	if alternativeImageProxy != "" && !disableAlternativeProxy {
		brand = replaceModAutorus(brand, false)
		number = replaceModAutorus(number, true)

		alternativeImageProxy = strings.TrimSuffix(alternativeImageProxy, "/")
		alternativeImageURL := alternativeImageProxy + "/images/" + strings.Replace(url.PathEscape(brand), "+", "%20", -1) + "/" + url.PathEscape(number) + "/full/" + imgName
		if updatePhoto || updatePhotoCount != 0 {
			updatePhotoCountStr := strconv.Itoa(updatePhotoCount)
			alternativeImageURL += "?" + updatePhotoCountStr
		}
		return alternativeImageURL
	}

	if strings.Contains(imgName, "http://") || strings.Contains(imgName, "https://") {
		return imgName
	}

	enc := encode4chars(imgName)
	imgName = addFileNamePostfix(enc, pfx)
	return "https://pubimg.nodacdn.net/images/" + imgName
}

// contactMethodConst содержит коды и текстовые значения способов обратной связи.
var contactMethodConst map[int]string = map[int]string{
	0: "По телефону и в сообщениях",
	1: "По телефону",
	2: "В сообщениях",
}

// buildContactMethod возвращает текстовое значение способа обратной связи по коду.
func buildContactMethod(m int) string {
	return contactMethodConst[m]
}

// adTypeConst содержит коды и текстовые значения типов объявлений.
var adTypeConst = map[int]string{
	0: "Товар приобретен на продажу",
	1: "Товар от производителя",
}

// buildAdType возвращает текстовое значение типа объявления по коду.
func buildAdType(t int) string {
	return adTypeConst[t]
}

// conditionConst содержит коды и текстовые значения условий.
var conditionConst map[int]string = map[int]string{
	0: "Новое",
	1: "Б/у",
}

// buildCondition возвращает текстовое значение условия по коду.
func buildCondition(c int) string {
	return conditionConst[c]
}

// adsConst содержит коды и текстовые статусы объявлений.
var adsConst map[int]string = map[int]string{
	0:  "Free",
	1:  "Premium",
	2:  "VIP",
	3:  "PushUp",
	4:  "Highlight",
	5:  "TurboSale",
	6:  "QuickSale",
	7:  "XL",
	8:  "x2_1",
	9:  "x2_7",
	10: "x5_1",
	11: "x5_7",
	12: "x10_1",
	13: "x10_7",
	14: "x15_1",
	15: "x15_7",
	16: "x20_1",
	17: "x20_7",
}

// buildAdStatus возвращает текстовое значение статуса объявления по коду.
func buildAdStatus(s int) string {
	return adsConst[s]
}

// feeConst содержит коды и текстовые статусы комиссии.
var feeConst = map[int]string{
	1: "Package",
	2: "PackageSingle",
	3: "Single",
}

// buildListingFee возвращает текстовое значение комиссии по коду.
func buildListingFee(fee int) string {
	return feeConst[fee]
}

// deleteDuplicateAddrs удаляет повторяющиеся адреса.
func deleteDuplicateAddrs(addresses []string) []string {

	unique := make(map[string]bool)
	var addrs []string
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if _, ok := unique[addr]; !ok {
			unique[addr] = true
			addrs = append(addrs, addr)
		}
	}

	return addrs
}

// getAddresses формирует список адресов для тега <Addresses>.
func getAddresses(addrs []string) *xmlOptions {

	opts := make([]xmlOption, 0)
	for _, addr := range addrs {
		addrRunes := []rune(addr)
		if len(addrRunes) == 0 || len(addrRunes) > 256 {
			continue
		}

		opts = append(opts, xmlOption{Value: addr})
		if len(opts) == 10 {
			break
		}
	}

	if len(opts) != 0 {
		return &xmlOptions{Options: opts}
	}

	return nil
}

// getRimType заменяет значение типа диска из article api
// на форму написания, указанную в документации Avito.
// Возвращает значение для тега <RimType>.
func getRimType(diskType string) string {

	switch diskType {
	case "литой":
		return "Литые"
	case "штампованный":
		return "Штампованные"
	case "кованый":
		return "Кованые"
	default:
		return ""
	}
}

// getSAEGearOils формирует значение тега <SAE> для группы товаров gear_oils.
func getSAEGearOils(prop any) string {

	switch val := prop.(type) {
	case string:
		if len(strings.TrimSpace(val)) != 0 {
			return val
		}

		return "Не подлежит классификации по SAE"
	case []string:
		return strings.Join(val, ",")
	default:
		return ""
	}
}

// getSAE формирует значение тега <SAE>.
func getSAE(prop any) string {

	switch val := prop.(type) {
	case string:
		return val
	case []string:
		return strings.Join(val, ",")
	default:
		return ""
	}
}

// getACEA формирует значение тега <ACEA>.
func getACEA(prop any) string {

	switch val := prop.(type) {
	case string:
		return val
	case []any:
		return val[0].(string)
	default:
		return ""
	}
}

// getDOT формирует значение тега <DOT>.
func getDOT(prop any) string {

	switch val := prop.(type) {
	case string:
		return val
	case []any:
		return val[0].(string)
	default:
		return ""
	}
}

// getPolarity формирует значение тега <Polarity>.
func getPolarity(prop any) string {

	switch prop.(string) {
	case "inverse", "обратная":
		return "Обратная"
	case "direct", "прямая":
		return "Прямая"
	case "universal", "универсальная":
		return "Двойная"
	default:
		return ""
	}
}

// getQuantity формирует значение тега <Quantity>.
func getQuantity(tiresQuantityType, tiresQuantity int, posPacking string) int {

	if tiresQuantityType == 0 {
		packing, _ := strconv.Atoi(posPacking)
		return packing
	}

	quantity := tiresQuantity
	if tiresQuantity <= 0 {
		quantity = 1
	}

	return quantity
}

// getAPISpec формирует значение тега <API>.
func getAPISpec(prop any) any {

	var res any
	switch val := prop.(type) {
	case []any:
		var options []string
		for _, specOpt := range val {
			options = append(options, specOpt.(string))
		}

		if len(options) > 1 {
			res = apiTagStructSlise{
				Option: options,
			}
		} else if len(options) == 1 {
			res = options[0]
		} else {
			res = nil
		}
	case string:
		res = val
	}

	return res
}

// getOEMOil формирует значение тега <OEMOil>.
func getOEMOil(prop any, avitoOemSpec map[string]string) []string {

	var options []string
	switch oemOpts := prop.(type) {
	case []any:
		for _, oemOpt := range oemOpts {
			abcpOEM := oemOpt.(string)
			if avitoOEM, ok := avitoOemSpec[abcpOEM]; ok {
				options = append(options, avitoOEM)
			} else {
				options = append(options, abcpOEM)
			}
		}
	}

	return options
}

// getInternetCalls формирует значение тега <InternetCalls>.
func getInternetCalls(internetCalls bool) string {

	if internetCalls {
		return "Да"
	}

	return "Нет"
}

// getTireYear формирует значение тега <TireYear>.
func (t currentTime) getTireYear(avitoTireYear string) string {

	if avitoTireYear == "" {
		return strconv.Itoa(t.AddDate(0, -6, 0).Year())
	}

	return avitoTireYear
}

// setOilsTags устанавливает значения в тегах для группы товаров oils.
func setOilsTags(offer xmlOffer, props map[string]any, number string) xmlOffer {

	if _, ok := props["viscosity"]; ok {
		offer.SAE = getSAE(props["viscosity"])
	}

	if _, ok := props["liquid_volume"]; ok {
		offer.Volume = replaceSeparatorToComma(props["liquid_volume"].(string)) + " л"
	}

	if _, ok := props["acea_spec"]; ok {
		offer.ACEA = getACEA(props["acea_spec"])
	}

	offer.VendorCode = number
	return offer
}

// setGearOilsTags устанавливает значения в тегах для группы товаров gear_oils.
func setGearOilsTags(offer xmlOffer, props map[string]any, number string) xmlOffer {

	if _, ok := props["viscosity"]; ok {
		offer.SAE = getSAEGearOils(props["viscosity"])
	} else {
		offer.SAE = "Не подлежит классификации по SAE"
	}

	if _, ok := props["liquid_volume"]; ok {
		offer.Volume = replaceSeparatorToComma(props["liquid_volume"].(string)) + " л"
	}

	offer.VendorCode = number
	return offer
}

// setBrakeFluidsTags устанавливает значения в тегах для группы товаров brake_fluids.
func setBrakeFluidsTags(offer xmlOffer, props map[string]any, number string) xmlOffer {

	if _, ok := props["dot_spec"]; ok {
		offer.DOT = getDOT(props["dot_spec"])
	}

	if _, ok := props["liquid_volume"]; ok {
		offer.Volume = replaceSeparatorToComma(props["liquid_volume"].(string)) + " л"
	}

	offer.VendorCode = number
	return offer
}

// setCoolantTags устанавливает значения в тегах для группы товаров coolant.
func setCoolantTags(offer xmlOffer, props map[string]any, number string) xmlOffer {

	if _, ok := props["coolant_color"]; ok {
		offer.Color = props["coolant_color"].(string)
	}

	if _, ok := props["liquid_volume"]; ok {
		offer.Volume = replaceSeparatorToComma(props["liquid_volume"].(string)) + " л"
	}

	offer.VendorCode = number
	return offer
}

// setBatteriesTags устанавливает значения в тегах для группы товаров batteries.
func setBatteriesTags(offer xmlOffer, props map[string]any) xmlOffer {

	if _, ok := props["voltage"]; ok {
		offer.Voltage = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(props["voltage"].(string),
			"V", ""), "В", ""), "V/В", "")
	}

	if _, ok := props["capacity"]; ok {
		offer.Capacity = props["capacity"].(string)
	}

	if _, ok := props["cca"]; ok {
		offer.DCL = props["cca"].(string)
	}

	if _, ok := props["polarity"]; ok {
		offer.Polarity = getPolarity(props["polarity"])
	}

	if _, ok := props["length"]; ok {
		offer.TechnicLength = props["length"].(string)
	}

	if _, ok := props["width"]; ok {
		offer.TechnicWidth = props["width"].(string)
	}

	if _, ok := props["height"]; ok {
		offer.TechnicHeight = props["height"].(string)
	}

	return offer
}

// setWipersTags устанавливает значения в тегах для группы товаров wipers.
func setWipersTags(offer xmlOffer, props map[string]any, brand string) xmlOffer {

	offer.InstallationLocation = "Лобовое стекло"

	if _, ok := props["pack_count"]; ok {
		offer.Set = getSet(props["pack_count"])
	}

	if _, ok := props["connector"]; ok {
		offer.MountingType = getMountingType(props["connector"])
	}

	if _, ok := props["construction"]; ok {
		caser := cases.Title(language.Russian)
		offer.BrushType = caser.String(props["construction"].(string))
	}

	if _, ok := props["length1"]; ok {
		offer.BrushLength = getBrushLength(props["length1"], "1")
	}

	if _, ok := props["length2"]; ok && offer.Set == "Да" {
		offer.SecondBrushLength = getBrushLength(props["length2"], "2")
	}

	offer.BrushBrand = brand
	return offer
}
