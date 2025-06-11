package main

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"gitlab.nodasoft.com/lib/gotypes"
)

type avitoParams struct {
	GoodsGroupsInPrice            []string          `json:"goodsGroupsInPrice"`
	Address                       string            `json:"address"`
	DisplayAreas                  []string          `json:"displayAreas"`
	ManagerName                   string            `json:"managerName"`
	ContactPhone                  string            `json:"contactPhone"`
	ContactMethod                 int               `json:"contactMethod"`
	DateEnd                       int               `json:"dateEnd"`
	AdType                        int               `json:"adType"`
	ListingFee                    int               `json:"listingFee"`
	AdStatus                      int               `json:"adStatus"`
	SalesConditions               string            `json:"salesConditions"`
	Condition                     int               `json:"condition"`
	PropertiesURL                 string            `json:"propertiesURL"`
	ExcludeOffersWithoutPicture   bool              `json:"excludeOffersWithoutPicture"`
	AlwaysGenerateImage           bool              `json:"alwaysGenerateImage"`
	DisableAlternativeImage       bool              `json:"disableAlternativeImage"`
	AvitoCategoriesList           map[string]string `json:"avitoCategoriesList"`
	DescrCategories               map[string]string `json:"descrCategories"`
	AlternativeImageProxy         string            `json:"alternativeImageProxy"`
	AlternativeImageRequestMethod string            `json:"alternativeImageRequestMethod"`
	Availability                  int               `json:"availability"`
	TiresQuantityType             int               `json:"tiresQuantityType"` // 0 - по умолчанию - берём значение из поля tiresQuantity, 1 - из прайса - берём значение packing из прайса
	TiresQuantity                 int               `json:"tiresQuantity"`     // если <=0, считаем, что задана "1"
	UpdateStockFormat             bool              `json:"updateStockFormat"`
	AvitoOfferID                  int               `json:"avitoOfferId"`
	L10nCategories                map[string]string `json:"l10nCategories"`
	VideoURL                      string            `json:"videoURL"`
	InternetCalls                 bool              `json:"internetCalls"`
	CallsDevices                  []string          `json:"callsDevices"`
	RemoveStatementTypeFromDescr  bool              `json:"removeStatementTypeFromDescr"`
	AdditionalAddresses           []string          `json:"additionalAddresses"`
	TireYear                      string            `json:"tireYear"`
	DeliveryFromPrices            []struct {
		MinPrice      float64  `json:"minPrice"`
		MaxPrice      float64  `json:"maxPrice"`
		DeliveryTypes []string `json:"deliveryTypes"`
	} `json:"DeliveryFromPrices"`
	HidePriceTag        bool `json:"hidePriceTag"`
	UpdatePhoto         bool `json:"updatePhoto"`
	UpdatePhotoCount    int  `json:"updatePhotoCount"`
	FilterOffersPicture int  `json:"filterOffersPicture"` // 0 - не использовать, 1 - Оставлять товары с изображениями, 2 - Оставлять товары без изображений
}

type avitoModelsStruct struct {
	Response []tireModel `json:"response"`
}

type tireModel struct {
	ID           int    `json:"id"`
	AvitoName    string `json:"avitoName"`
	AvitoNameFix string `json:"avitoNameFix"`
	AbcpName     string `json:"abcpName"`
	Manual       bool   `json:"manual"`
}

type avitoCategoriesTagsStruct struct {
	Category             string `json:"category"`
	GoodsType            string `json:"goodsType"`
	ProductType          string `json:"productType"`
	SparePartType        string `json:"sparePartType"`
	TechnicSparePartType string `json:"technicSparePartType"`
	GoodsGroup           string `json:"goodsGroup"`
	SparePartType2       string `json:"sparePartType2"`
}

var errNoParams = errors.New("отсутствуют параметры для заданого типа прайса")

func createFileAndWriteHeader() (string, error) {
	fileName := tempPath + gotypes.NewUUID() + ".xml"
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	header := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Ads formatVersion=\"3\" target=\"Avito.ru\">\n"
	if _, err := file.WriteString(header); err != nil {
		return "", err
	}

	return fileName, nil
}

func createFileAndWriteHeaderStock() (string, error) {
	fileName := tempPath + gotypes.NewUUID() + ".xml"
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	timeNowWithFormat := time.Now().Format("2006-01-02T15:04:05")
	header := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<items date=\"" + timeNowWithFormat + "\" formatVersion=\"1\" class=\" FB_FW_ext Bco\">\n"
	if _, err := file.WriteString(header); err != nil {
		return "", err
	}

	return fileName, nil
}

func (s *offersTaskStock) processXMLStock(inputChan <-chan []position, avito *avitoParams, params Params,
	goodsGroupPack map[string]map[string]map[string]string, regexpsIncludedDesc, regexpsExcludedDesc []*regexp.Regexp) {

	defer close(s.output)

	var pns map[string]string
	offers := make(map[string]xmlOfferStock)
	pfx := "0005"
	properties, err := parseProperties(avito.PropertiesURL)
	if err != nil {
		s.err = err
		return
	}
	for positions := range inputChan {
		for _, pos := range positions {
			var offerImages []xmlImage
			pns = goodsGroupPack[pos.GoodsGroupCode]["pns"]
			key := strings.ToUpper(pos.Brand + "|" + pos.Number)
			props := copyMap(properties[key])

			offerID := buildOfferID(pos, avito.AvitoOfferID)

			offer := xmlOfferStock{
				ID:    offerID,
				Stock: pos.Availability,
			}

			var xmlParams []xmlParam
			if pos.GoodsGroupCode == "" || pos.GoodsGroupCode == "others" {
				if props["goods_group"] != nil {
					xmlParams = getXMLParams(props, pns, params.PriorityDescriptionSource)
				}
			} else {
				xmlParams = getXMLParams(props, pns, params.PriorityDescriptionSource)
			}

			if pos.GoodsGroupCode == "wipers" || props["goods_group"] == "wipers" {
				xmlParams = append(xmlParams, addWipersParams(pos.Brand)...)
			}

			productDetail := buildProductDetail(xmlParams)

			desc := buildOfferDescription(pos, productDetail, params.PriorityDescriptionSource, params.Localization, avito.RemoveStatementTypeFromDescr)
			if excludeOfferWithDescriptions(params.IncludedDescriptions, params.ExcludedDescriptions, desc, regexpsIncludedDesc, regexpsExcludedDesc) {
				continue
			}

			if imgProp, ok := props["images"]; ok {
				if imgs, ok := imgProp.([]any); ok && len(imgs) != 0 {
					for _, img := range imgs {

						imgURL := buildImgURLOffer(img.(string), pfx, avito.AlternativeImageProxy, pos.Brand, pos.Number,
							avito.DisableAlternativeImage, avito.UpdatePhoto, avito.UpdatePhotoCount)
						imgURL = addImageIncParam(imgURL, params.ImageInc)
						offerImages = append(offerImages, xmlImage{URL: imgURL})

						if avito.AlternativeImageProxy != "" {
							break
						}

						if len(offerImages) == 10 {
							break
						}
					}
				}
			}

			if len(offerImages) == 0 && avito.AlternativeImageProxy != "" && avito.AlwaysGenerateImage {
				emptyImg := "05c40c050e1eeef58efb8bcf8e6ce2510b.png"
				if avito.AlternativeImageRequestMethod == "URL-JPG" {
					emptyImg = "1149f97a082eb731bab1e4d0bb281be3e8.jpg"
				}

				imgURL := strings.TrimSuffix(avito.AlternativeImageProxy, "/") +
					"/images/" + strings.ToLower(replaceModAutorus(pos.Brand, false)) + "/" + strings.ToLower(replaceModAutorus(pos.Number, true)) +
					"/full/" + emptyImg
				imgURL = addImageIncParam(imgURL, params.ImageInc)

				offerImages = append(offerImages, xmlImage{imgURL})
			}

			// Блок для работы с бу товарами
			if pos.Condition != 0 {
				offerImages = make([]xmlImage, 0)
				for _, img := range pos.UsedImages {

					imgURL := buildImgURLOffer(img, pfx, avito.AlternativeImageProxy, pos.Brand, pos.Number, avito.DisableAlternativeImage, avito.UpdatePhoto, avito.UpdatePhotoCount)
					imgURL = addImageIncParam(imgURL, params.ImageInc)
					offerImages = append(offerImages, xmlImage{URL: imgURL})

					if avito.AlternativeImageProxy != "" {
						break
					}

					if len(offerImages) == 10 {
						break
					}
				}
			}
			if len(offerImages) == 0 && avito.ExcludeOffersWithoutPicture {
				continue
			}
			offers[offer.ID] = offer
		}
		var content []xmlOfferStock
		for _, v := range offers {
			content = append(content, v)
		}
		offers = make(map[string]xmlOfferStock)
		s.count += len(offers)
		s.output <- content
	}
}

func (s *offersTask) processXML(posChan <-chan []position, avito *avitoParams, params Params,
	goodsGroupPack map[string]map[string]map[string]string, regexpsIncludedDescription, regexpsExcludedDescription []*regexp.Regexp) {

	defer close(s.output)

	var (
		pns           map[string]string
		propTranslate map[string]string
		avitoModels   avitoModelsStruct
	)

	if avito == nil {
		s.err = errNoParams
		return
	}

	regexpTiresModel, err := regexp.Compile("[^a-zA-Zа-яА-ЯёЁ0-9]")
	if err != nil {
		log.Errorf("Ошибка компиляции регулярного выражения для моделей шин")
	}

	offers := make(map[string]xmlOffer)

	properties, err := parseProperties(avito.PropertiesURL)
	if err != nil {
		s.err = err
		return
	}
	avitoCategoriesTags := getAvitoCategoriesTags()
	avitoDescrCategoriesTags := getAvitoDescrCategoriesTags()
	avitoTruckDescrCategoriesTags := getAvitoTruckDescrCategoriesTags()
	avitoBrandCategoriesTags := getAvitoBrandCategoriesTags()
	avitoOemSpec := getAvitoSpec("OemSpec", "oem_spec")
	avitoAtfSpec := getAvitoSpec("ATF", "atf_spec")

	pfx := "0005"

	for positions := range posChan {
		for _, pos := range positions {

			propTranslate = goodsGroupPack[pos.GoodsGroupCode]["translatedprops"]
			offerID := buildOfferID(pos, avito.AvitoOfferID)

			key := strings.ToUpper(pos.Brand + "|" + pos.Number)
			props := copyMap(properties[key])
			if pos.GoodsGroupCode == "" {
				if _, ok := props["goods_group"]; ok {
					pos.GoodsGroupCode = props["goods_group"].(string)
				}

			}

			pns = goodsGroupPack[pos.GoodsGroupCode]["pns"]
			for name, val := range props {
				switch props[name].(type) {
				case string:
					if _, ok := propTranslate[props[name].(string)]; ok && propTranslate[props[name].(string)] != "" {
						props[name] = propTranslate[props[name].(string)]
					} else {
						props[name] = val
					}
				case []any:
					var translArr []any
					for _, va := range props[name].([]any) {
						if ss, ok := propTranslate[va.(string)]; ok {
							if ss != "" {
								translArr = append(translArr, ss)
							} else {
								translArr = append(translArr, va)
							}
						} else {
							translArr = append(translArr, va)
						}

					}
					props[name] = translArr
				default:
					props[name] = val
				}
			}

			var offer xmlOffer
			price := int(math.Ceil(pos.PriceSale))

			if pos.isTires() {
				if len(avitoModels.Response) == 0 {
					avitoModels = getAvitoModels()
				}

				if pos.Condition != 0 {
					curTime := currentTime{time.Now()}
					offer.TireYear = curTime.getTireYear(avito.TireYear)
				}

				offer.Model = buildOfferModel(getString(props["catalog_model"]), avitoModels, regexpTiresModel)
			}

			offer.ID = offerID
			if len([]rune(avito.Address)) > 256 {
				s.err = fmt.Errorf("address more than 256 characters")
				return
			}
			offer.Address = avito.Address

			if len(avito.AdditionalAddresses) != 0 {
				avito.AdditionalAddresses = deleteDuplicateAddrs(avito.AdditionalAddresses)
				offer.Addresses = getAddresses(avito.AdditionalAddresses)
			}

			if len(avito.DisplayAreas) > 0 {
				displayAreas := make([]string, 0)
				displayAreas = append(displayAreas, avito.DisplayAreas...)
				offer.DisplayAreas = &displayAreas
			}

			offer.ContactPhone = avito.ContactPhone

			if len([]rune(avito.ManagerName)) > 40 {
				s.err = fmt.Errorf("manager Name more than 40 characters")
				return
			}
			offer.ManagerName = avito.ManagerName

			if pos.isDisks() {
				offer.RimBrand = pos.Brand
			} else {
				offer.Brand = pos.Brand
			}

			offer.DateEnd = buildDateEnd(time.Now(), avito.DateEnd)
			offer.ListingFee = buildListingFee(avito.ListingFee)
			offer.AdStatus = buildAdStatus(avito.AdStatus)
			offer.ContactMethod = buildContactMethod(avito.ContactMethod)
			offer.AdType = buildAdType(avito.AdType)
			offer.Condition = buildCondition(avito.Condition)

			if imgProp, ok := props["images"]; ok {
				if imgs, ok := imgProp.([]any); ok && len(imgs) != 0 {
					for _, img := range imgs {

						imgURL := buildImgURLOffer(img.(string), pfx, avito.AlternativeImageProxy, pos.Brand, pos.Number, avito.DisableAlternativeImage, avito.UpdatePhoto, avito.UpdatePhotoCount)
						imgURL = addImageIncParam(imgURL, params.ImageInc)
						offer.Images = append(offer.Images, xmlImage{URL: imgURL})

						if avito.AlternativeImageProxy != "" {
							break
						}

						if len(offer.Images) == 10 {
							break
						}
					}
				}
			}

			if len(offer.Images) == 0 && avito.AlternativeImageProxy != "" && avito.AlwaysGenerateImage {
				emptyImg := "05c40c050e1eeef58efb8bcf8e6ce2510b.png"
				if avito.AlternativeImageRequestMethod == "URL-JPG" {
					emptyImg = "1149f97a082eb731bab1e4d0bb281be3e8.jpg"
				}

				imgURL := strings.TrimSuffix(avito.AlternativeImageProxy, "/") +
					"/images/" + strings.ToLower(replaceModAutorus(pos.Brand, false)) + "/" + strings.ToLower(replaceModAutorus(pos.Number, true)) +
					"/full/" + emptyImg
				imgURL = addImageIncParam(imgURL, params.ImageInc)

				offer.Images = append(offer.Images, xmlImage{imgURL})
			}

			// Блок для работы с бу товарами
			if pos.Condition != 0 {
				offer.Images = make([]xmlImage, 0)
				for _, img := range pos.UsedImages {

					imgURL := buildImgURLOffer(img, pfx, avito.AlternativeImageProxy, pos.Brand, pos.Number, avito.DisableAlternativeImage, avito.UpdatePhoto, avito.UpdatePhotoCount)
					imgURL = addImageIncParam(imgURL, params.ImageInc)
					offer.Images = append(offer.Images, xmlImage{URL: imgURL})

					if avito.AlternativeImageProxy != "" {
						break
					}

					if len(offer.Images) == 10 {
						break
					}
				}

				offer.Condition = params.Localization.WearoutPreOwned
			}

			if len(offer.Images) == 0 && avito.ExcludeOffersWithoutPicture {
				continue
			}

			switch avito.FilterOffersPicture {
			case 1:
				if len(offer.Images) == 0 {
					continue
				}
			case 2:
				if len(offer.Images) != 0 {
					continue
				}
			}

			offer.VideoURL = avito.VideoURL

			if pos.Description == "" {
				if descriptionProp, ok := props["descr"]; ok {
					if description, ok := descriptionProp.(string); ok {
						pos.Description = description
					}
				}
			} else {
				pos.Description = regexpDescription.ReplaceAllString(pos.Description, "")
			}

			var xmlParams []xmlParam
			if pos.GoodsGroupCode == "" || pos.GoodsGroupCode == "others" {
				if props["goods_group"] != nil {
					xmlParams = getXMLParams(props, pns, params.PriorityDescriptionSource)
				}
			} else {
				xmlParams = getXMLParams(props, pns, params.PriorityDescriptionSource)
			}

			if pos.GoodsGroupCode == "wipers" || props["goods_group"] == "wipers" {
				xmlParams = append(xmlParams, addWipersParams(pos.Brand)...)
			}

			productDetail := buildProductDetail(xmlParams)

			description := buildOfferDescription(pos, productDetail, params.PriorityDescriptionSource, params.Localization, avito.RemoveStatementTypeFromDescr)
			if excludeOfferWithDescriptions(params.IncludedDescriptions, params.ExcludedDescriptions, description, regexpsIncludedDescription, regexpsExcludedDescription) {
				continue
			}

			description = buildFinalOfferDescription(description, avito.SalesConditions)
			offer.Description = newCharData(description)

			excludeReqCategoryByBrandForGoodsGroups := map[string]bool{
				"22":  true,
				"105": true,
				"26":  true,
				"33":  true,
				"9":   true,
				"81":  true,
				"10":  true,
				"76":  true,
				"75":  true,
				"109": true,
				"106": true,
				"214": true,
				"110": true,
			}
			ggID := getGoodsGroupsID(props["goods_group"], pos.GoodsGroupCode)
			//if !excludeReqCategoryByBrandForGoodsGroups[ggID] && pos.ArticlesIsСargo(avitoBrandCategoriesTags, params.PriorityDescriptionSource, offer.Brand, avitoDescrCategoriesTags, avitoCategoriesTags, ggID) {
			//	offer.ProductType = "Для грузовиков и спецтехники"
			//}

			// Сначала определяем категорию по бренду
			if !excludeReqCategoryByBrandForGoodsGroups[ggID] {
				brandCategory, brandGoodsType, brandProductType, brandSparePartType := pos.buildTagsByBrand(avitoBrandCategoriesTags, params.PriorityDescriptionSource, offer.Brand)
				offer.ProductType = brandProductType
				offer.SparePartType = brandSparePartType
				offer.Category = brandCategory
				offer.GoodsType = brandGoodsType
			}

			// Определяем категорию по goodsGroup если не заполнили по бренду
			var sparePartType2 string
			if offer.ProductType == "" && offer.SparePartType == "" && offer.Category == "" && offer.GoodsType == "" {
				if offer.ProductType == "" {
					offer.ProductType = avitoCategoriesTags[ggID].ProductType
				}
				if offer.SparePartType == "" {
					offer.SparePartType = avitoCategoriesTags[ggID].SparePartType
				}
				if offer.Category == "" {
					offer.Category = avitoCategoriesTags[ggID].Category
				}
				if offer.GoodsType == "" {
					offer.GoodsType = avitoCategoriesTags[ggID].GoodsType
				}

				sparePartType2 = avitoCategoriesTags[ggID].SparePartType2
			}

			// Определяем категорию по описанию если не заполнили по бренду и по goodsGroup
			if offer.ProductType == "" && offer.SparePartType == "" && offer.Category == "" && offer.GoodsType == "" && sparePartType2 == "" {
				descrCategory, descrGoodsType, descrProductType, descrSparePartType, descrGoodsGroup, descrSparePartType2 := pos.buildTagsByDescription(avitoCategoriesTags, avitoDescrCategoriesTags, params.PriorityDescriptionSource)
				if offer.ProductType == "" {
					offer.ProductType = descrProductType
				}
				if offer.SparePartType == "" {
					offer.SparePartType = descrSparePartType
				}
				if offer.Category == "" {
					offer.Category = descrCategory
				}
				if offer.GoodsType == "" {
					offer.GoodsType = descrGoodsType
				}

				if sparePartType2 == "" {
					sparePartType2 = descrSparePartType2
				}

				props["goods_group"] = descrGoodsGroup
			}

			if offer.ProductType == "" && offer.SparePartType == "" && offer.Category == "" && offer.GoodsType == "" && sparePartType2 == "" {
				ggID := "1"
				if offer.ProductType == "" {
					offer.ProductType = avitoCategoriesTags[ggID].ProductType
				}
				if offer.SparePartType == "" {
					offer.SparePartType = avitoCategoriesTags[ggID].SparePartType
				}
				if offer.Category == "" {
					offer.Category = avitoCategoriesTags[ggID].Category
				}
				if offer.GoodsType == "" {
					offer.GoodsType = avitoCategoriesTags[ggID].GoodsType
				}

				if sparePartType2 == "" {
					sparePartType2 = avitoCategoriesTags[ggID].SparePartType2
				}
			}

			if offer.ProductType == "Для грузовиков и спецтехники" {
				sparePartType, technicSparePartType := pos.buildTagsByTruckDescription(avitoTruckDescrCategoriesTags, params.PriorityDescriptionSource)

				offer.SparePartType = sparePartType
				offer.TechnicSparePartType = technicSparePartType
				if offer.SparePartType == "" && offer.TechnicSparePartType == "" {
					offer.SparePartType = "Трансмиссия"
					offer.TechnicSparePartType = "Детали КПП"
				}
			}

			if offer.ProductType == "Трансмиссионные масла" {
				applicabilityProp := getApplicabilityProp(props)
				if applicabilityProp == "ГУР" {
					offer.ProductType = "Гидравлические жидкости"
				}
			}

			if offer.GoodsType == "Аксессуары" {
				offer.AccessoryType = offer.SparePartType
				offer.SparePartType = ""
			}

			if offer.GoodsType == "Противоугонные устройства" {
				offer.DeviceType = offer.ProductType
				offer.ProductType = ""
			}

			if offer.AccessoryType == "Дефлекторы" {
				offer.InstallationLocation = "Окна"
			}

			// Построение специфических тегов для определенных goodsGroups
			if pos.GoodsGroupCode == "bicycles" || props["goods_group"] == "bicycles" {
				offer.VehicleType = buildVehicleType(props)
			}

			if pos.GoodsGroupCode == "gear_oils" || props["goods_group"] == "gear_oils" {
				if _, ok := props["atf_spec"]; ok {
					abcpATF := getAbcpATFByType(props["atf_spec"])
					offer.ATF = getATF(avitoAtfSpec, abcpATF)
				}
			}

			if pos.GoodsGroupCode == "compressor_oils" || props["goods_group"] == "compressor_oils" {
				if _, ok := props["liquid_volume"]; ok {
					offer.Volume = replaceSeparatorToComma(props["liquid_volume"].(string)) + " л"
				}
			}

			if pos.GoodsGroupCode == "oils" || props["goods_group"] == "oils" {
				offer = setOilsTags(offer, props, pos.Number)
			}

			if pos.GoodsGroupCode == "gear_oils" || props["goods_group"] == "gear_oils" {
				offer = setGearOilsTags(offer, props, pos.Number)
			}

			if pos.GoodsGroupCode == "brake_fluids" || props["goods_group"] == "brake_fluids" {
				offer = setBrakeFluidsTags(offer, props, pos.Number)
			}

			if pos.GoodsGroupCode == "coolant" || props["goods_group"] == "coolant" {
				offer = setCoolantTags(offer, props, pos.Number)
			}

			if pos.GoodsGroupCode == "batteries" || props["goods_group"] == "batteries" {
				offer = setBatteriesTags(offer, props)
			}

			if pos.GoodsGroupCode == "wipers" || props["goods_group"] == "wipers" {
				offer = setWipersTags(offer, props, pos.Brand)
			}

			if offer.SparePartType == "Кузов" {
				offer.BodySparePartType = sparePartType2
			}

			if offer.SparePartType == "Двигатель" {
				offer.EngineSparePartType = sparePartType2
			}

			if offer.GoodsType == "Багажники и фаркопы" {
				offer.TrunkType = sparePartType2
			}

			if offer.SparePartType == "Трансмиссия и привод" {
				offer.TransmissionSparePartType = sparePartType2
			}

			offer.Title = buildOfferName(pos, props, params.Localization, params.PriorityDescriptionSource)

			if pos.GoodsGroupCode == "gear_oils" || props["goods_group"] == "gear_oils" ||
				pos.GoodsGroupCode == "oils" || props["goods_group"] == "oils" {
				offer.API = getAPISpec(props["api_spec"])
			}

			if pos.isTires() {
				offer.Quantity = getQuantity(avito.TiresQuantityType, avito.TiresQuantity, pos.Packing)
				price = price * offer.Quantity
			}

			offer.Availability = buildAvailability(avito.Availability, pos.DeadLine)

			if props["goods_group"] == "wheel_covers" {
				if v, ok := props["diameter"]; ok {
					if str, ok := v.(string); ok {
						offer.RimDiameter = str
					}
				}
			}

			if pos.isTires() || pos.isDisks() {
				if v, ok := props["axle"]; ok {
					if str, ok := v.(string); ok {
						offer.WheelAxle = str
					}
				}

				if pos.GoodsGroupCode == "tires" {
					if v, ok := props["season"]; ok {
						if season, ok := v.(string); ok {
							offer.TireType = getTyreType(season)
						}
					}
				}

				if pos.GoodsGroupCode == "truck_tires" {
					offer.TireType = "Всесезонные"
				}

				if v, ok := props["diameter"]; ok {
					if str, ok := v.(string); ok {
						offer.RimDiameter = str
					}
				}

				if v, ok := props["axle"]; ok {
					if str, ok := v.(string); ok {
						offer.WheelAxle = str
					}
				}

				if v, ok := props["disk_type"]; ok {
					if str, ok := v.(string); ok {
						offer.RimType = getRimType(str)
					}
				}

				if v, ok := props["holes"]; ok {
					if str, ok := v.(string); ok {
						offer.RimBolts = str
					}
				}

				if v, ok := props["pcd"]; ok {
					if str, ok := v.(string); ok {
						offer.RimBoltsDiameter = str
					}
				}

				if v, ok := props["et"]; ok {
					if str, ok := v.(string); ok {
						offer.RimOffset = str
					}
				}
			}

			if pos.isTires() {
				if v, ok := props["width"]; ok {
					if str, ok := v.(string); ok {
						offer.TireSectionWidth = str
					}
				}
				if v, ok := props["height"]; ok {
					if str, ok := v.(string); ok {
						offer.TireAspectRatio = str
					}
				}

				if offer.TireAspectRatio == "0" || offer.TireSectionWidth == "0" {
					continue
				}
			}

			if pos.isDisks() {
				if v, ok := props["width"]; ok {
					if str, ok := v.(string); ok {
						offer.RimWidth = str
					}
				}
				if v, ok := props["hub_diameter"]; ok {
					if str, ok := v.(string); ok {
						offer.RimDia = str
					}
				}
				if offer.RimWidth == "0" {
					continue
				}
			}

			if !pos.isTires() && !pos.isDisks() && !pos.isOils() && (pos.GoodsGroupCode != "bicycles" || props["goods_group"] != "bicycles") {
				offer.OEM = pos.Number
			}

			if pos.isOils() {

				var ok bool

				if pos.GoodsGroupCode == "coolant" {
					if _, ok = properties[key]["coolant_oem_spec"]; ok {
						options := getOEMOil(properties[key]["coolant_oem_spec"], avitoOemSpec)
						if len(options) > 0 {
							offer.OEMOil = &options
						}
					}
				} else {
					if _, ok = properties[key]["oem_spec"]; ok {
						options := getOEMOil(properties[key]["oem_spec"], avitoOemSpec)
						if len(options) > 0 {
							offer.OEMOil = &options
						}
					}
				}

				if pos.GoodsGroupCode == "coolant" {
					if _, ok = properties[key]["coolant_astm_spec"]; ok {
						astm, ok := properties[key]["coolant_astm_spec"].([]any)
						if ok && len(astm) > 0 {
							var aa []string
							for _, v := range astm {
								aa = append(aa, v.(string))
							}
							offer.ASTM = &aa
						}
					}
				}
			}

			offer.InternetCalls = getInternetCalls(avito.InternetCalls)

			if len(avito.CallsDevices) > 0 {
				dd := make([]string, 0)
				dd = append(dd, avito.CallsDevices...)
				offer.CallsDevices = &dd
			}

			if len(avito.DeliveryFromPrices) > 0 {
				for _, delivery := range avito.DeliveryFromPrices {
					// если цена товара попадает в диапазон между MinPrice и MaxPrice
					// или больше MinPrice, когда MaxPrice не указан,
					// создаём тег Delivery с перечнем возможных доставок
					if price >= int(delivery.MinPrice) &&
						(price < int(delivery.MaxPrice) || delivery.MaxPrice == 0) {

						offer.Delivery = &delivery.DeliveryTypes
						break
					}
				}
			}

			if !avito.HidePriceTag {
				offer.Price = price
			}

			offers[offer.ID] = offer
		}

		var content []xmlOffer
		for _, v := range offers {
			content = append(content, v)
		}

		offers = make(map[string]xmlOffer)
		s.count += len(content)
		s.output <- content
	}

	log.Printf("Задача %d: определены офферы для %d позиций", s.id, s.count)
}

func excludeOfferWithDescriptions(includedDesc, excludedDesc []string, desc string, regexpsIncludedDesc, regexpsExcludedDesc []*regexp.Regexp) bool {

	var isExclude bool

	if len(regexpsIncludedDesc) > 0 {
		for i, re := range regexpsIncludedDesc {
			if re.MatchString(desc) {
				isExclude = false
				break
			}

			if i == len(regexpsIncludedDesc)-1 {
				isExclude = true
			}
		}
	}

	if len(regexpsExcludedDesc) > 0 {
		for _, re := range regexpsExcludedDesc {
			if re.MatchString(desc) {
				isExclude = true
			}
		}
	}

	if len(excludedDesc) > 0 {
		desc = strings.ToLower(desc)
		for _, excluded := range excludedDesc {
			if strings.Contains(desc, strings.ToLower(excluded)) {
				isExclude = true
				return isExclude
			}
		}
	}

	if len(includedDesc) > 0 {
		desc = strings.ToLower(desc)
		for i, included := range includedDesc {
			if strings.Contains(desc, strings.ToLower(included)) {
				isExclude = false
				break
			}

			if i == len(includedDesc)-1 {
				isExclude = true
			}
		}
	}

	return isExclude
}

func writeOffersInFile(inputchan <-chan []xmlOffer, fileName string) (string, int, error) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return "", 0, errors.Errorf("Не получилось открыть файл: %v", err)
	}
	defer file.Close()
	count := 0
	for offers := range inputchan {
		output, err := xml.MarshalIndent(offers, "  ", "    ")
		if err != nil {
			return "", 0, errors.Errorf("Ошибка маршаллинга позиций: %v", err)
		}
		if _, err := file.Write(output); err != nil {
			return "", 0, errors.Errorf("Ошибка записи позиций в файл: %v", err)
		}
		count += len(offers)
	}
	if _, err := file.WriteString("\n</Ads>"); err != nil {
		return "", 0, errors.Errorf("Ошибка записи позиций в файл: %v", err)
	}
	return fileName, count, nil
}

func writeOffersStockInFile(inputchan <-chan []xmlOfferStock, fileName string) (string, int, error) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return "", 0, errors.Errorf("Не получилось открыть файл: %v", err)
	}
	defer file.Close()
	count := 0
	for offers := range inputchan {
		output, err := xml.MarshalIndent(offers, "  ", "    ")
		if err != nil {
			return "", 0, errors.Errorf("Ошибка маршаллинга позиций: %v", err)
		}
		if _, err := file.Write(output); err != nil {
			return "", 0, errors.Errorf("Ошибка записи позиций в файл: %v", err)
		}
		count += len(offers)
	}

	if _, err := file.WriteString("\n</items>"); err != nil {
		return "", 0, errors.Errorf("Ошибка записи позиций в файл: %v", err)
	}
	return fileName, count, nil
}

func getXMLParams(props Properties, pns map[string]string, priorityDescSource int) []xmlParam {

	requiredPropsByGoodsGroup := map[string][]string{
		"bicycles":        {"age", "type"},
		"gear_oils":       {"atf_spec", "viscosity", "liquid_volume", "api_spec"},
		"compressor_oils": {"liquid_volume"},
		"oils":            {"viscosity", "liquid_volume", "acea_spec", "api_spec"},
		"brake_fluids":    {"dot_spec", "liquid_volume"},
		"coolant":         {"coolant_color", "liquid_volume"},
		"batteries":       {"voltage", "capacity", "cca", "polarity", "length", "width", "height"},
		"wheel_covers":    {"diameter"},
		"moto_tires":      {"axle", "diameter", "disk_type", "holes", "pcd", "et", "width", "height"},
		"truck_tires":     {"axle", "season", "diameter", "disk_type", "holes", "pcd", "et", "width", "height"},
		"tires":           {"axle", "season", "diameter", "disk_type", "holes", "pcd", "et", "width", "height"},
		"disks":           {"axle", "season", "diameter", "disk_type", "holes", "pcd", "et", "width", "hub_diameter"},
		"wipers":          {"pack_count", "connector", "construction", "length1", "length2"},
	}

	var group string
	requiredProps := make([]string, 0)
	CopyPropsForXMLParams := copyMap(props)
	gg := CopyPropsForXMLParams["goods_group"]
	if gg != nil {
		group = gg.(string)
		requiredProps = requiredPropsByGoodsGroup[group]
	}

	if len(requiredProps) != 0 && (priorityDescSource == 0 || priorityDescSource == 1 || priorityDescSource == 3 || priorityDescSource == 4) {
		for currentProp := range CopyPropsForXMLParams {

			for i, requiredProp := range requiredProps {
				if currentProp == requiredProp {
					break
				}

				if i == len(requiredProps)-1 {
					delete(CopyPropsForXMLParams, currentProp)
				}
			}
		}
	}

	out := make([]xmlParam, 0)
	for name, value := range CopyPropsForXMLParams {

		translated, ok := pns[name]
		if !ok {
			continue
		}

		_, isSliceAny := value.([]any)
		if group == "wipers" && name == "connector" && isSliceAny {
			value = value.([]any)[0]
		}

		if sliceVal := reflect.ValueOf(value); sliceVal.Kind() == reflect.Slice {
			slice := make([]string, 0)
			for i := 0; i < sliceVal.Len(); i++ {
				if str, ok := sliceVal.Index(i).Interface().(string); ok {
					slice = append(slice, str)
				}
			}

			out = append(out, xmlParam{
				Name:    translated,
				Content: strings.Join(slice, "/"),
			})

			continue
		}

		var content string
		switch v := value.(type) {
		case float64:
			content = strconv.FormatFloat(v, 'f', 2, 64)
		case string:
			switch name {
			case "liquid_volume":
				content = replaceSeparatorToComma(v)
			case "connector":
				content = getMountingType(value)
			case "construction":
				caser := cases.Title(language.Russian)
				content = caser.String(v)
			default:
				content = v
			}
		case int:
			content = strconv.Itoa(v)
		default:
			log.Warningf("Неизвестный тип %T: %v (свойство %s)", v, v, translated)
			continue
		}

		if content != "" {
			out = append(out, xmlParam{
				Name:    translated,
				Content: content,
			})
		}
	}

	return out
}

func addFileNamePostfix(name, pfx string) string {

	ss := strings.Split(name, ".")
	l := len(ss)

	if l == 0 {
		return pfx
	}

	if l == 1 {
		return name + pfx
	}

	ss[l-2] = ss[l-2] + pfx

	return strings.Join(ss, ".")
}

func (pos *position) ArticlesIsСargo(brands map[string]avitoCategoriesTagsStruct, priorityDescriptionSource int, posBrand string, descrCategories, avitoCategoriesTags map[string]avitoCategoriesTagsStruct, ggID string) bool {

	var (
		productType        string
		matchInDescription bool
	)
	if avitoCategoriesTags[ggID].ProductType == "Для грузовиков и спецтехники" {
		return true
	}
	for brand, tags := range brands {
		switch priorityDescriptionSource {
		case 2:
			if pos.AdditionalDescription == "" {
				matchInDescription = caseInsensitiveContains(pos.Description, brand)

				if matchInDescription && len(getWordsFrom(brand)) == 1 {
					words := getWordsFrom(pos.Description)
					for _, word := range words {
						if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
							break
						}
					}
				}
				if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
					if productType == "Для грузовиков и спецтехники" {
						return true
					}
				}
			} else {
				matchInDescription = caseInsensitiveContains(pos.AdditionalDescription, brand)
				if matchInDescription && len(getWordsFrom(brand)) == 1 {
					words := getWordsFrom(pos.AdditionalDescription)
					for _, word := range words {
						if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
							break
						}
					}
				}
				if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
					productType = tags.ProductType
					if productType == "Для грузовиков и спецтехники" {
						return true
					}
				}
			}
		case 3:
			if pos.AdditionalDescription == "" {

				matchInDescription = caseInsensitiveContains(pos.Description, brand)
				if matchInDescription && len(getWordsFrom(brand)) == 1 {
					words := getWordsFrom(pos.Description)
					for _, word := range words {
						if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
							break
						}
					}
				}
				if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
					productType = tags.ProductType
					if productType == "Для грузовиков и спецтехники" {
						return true
					}
				}
			} else {
				matchInDescription = caseInsensitiveContains(pos.AdditionalDescription, brand)
				if matchInDescription && len(getWordsFrom(brand)) == 1 {
					words := getWordsFrom(pos.AdditionalDescription)
					for _, word := range words {
						if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
							break
						}
					}
				}
				if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
					productType = tags.ProductType
					if productType == "Для грузовиков и спецтехники" {
						return true
					}
				}
			}
		default:
			matchInDescription = caseInsensitiveContains(pos.Description, brand)

			if matchInDescription && len(getWordsFrom(brand)) == 1 {
				words := getWordsFrom(pos.Description)
				for _, word := range words {
					if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
						break
					}
				}
			}
			if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
				productType = tags.ProductType
				if productType == "Для грузовиков и спецтехники" {
					return true
				}
			}
		}
	}

	for descr, tags := range descrCategories {
		switch priorityDescriptionSource {
		case 2, 3:

			if pos.AdditionalDescription == "" {

				matchInDescription = caseInsensitiveContains(pos.Description, descr)
				if matchInDescription && len(getWordsFrom(descr)) == 1 {
					descrWords := getWordsFrom(pos.Description)
					for _, descrWord := range descrWords {
						if matchInDescription = strings.HasPrefix(strings.ToUpper(descrWord), strings.ToUpper(getWordsFrom(descr)[0])); matchInDescription {
							break
						}
					}
				}
				if matchInDescription {
					productType = tags.ProductType
					if productType == "Для грузовиков и спецтехники" {
						return true
					}
				}
			} else {
				matchInDescription = caseInsensitiveContains(pos.AdditionalDescription+" "+pos.Description, descr)
				if matchInDescription && len(getWordsFrom(descr)) == 1 {
					descrWords := getWordsFrom(pos.AdditionalDescription + " " + pos.Description)
					for _, descrWord := range descrWords {
						if matchInDescription = strings.HasPrefix(strings.ToUpper(descrWord), strings.ToUpper(getWordsFrom(descr)[0])); matchInDescription {
							break
						}
					}
				}

				if matchInDescription {
					productType = tags.ProductType
					if productType == "Для грузовиков и спецтехники" {
						return true
					}
				}
			}
		default:
			matchInDescription = caseInsensitiveContains(pos.Description, descr)

			if matchInDescription && len(getWordsFrom(descr)) == 1 {
				descrWords := getWordsFrom(pos.Description)
				for _, descrWord := range descrWords {
					if matchInDescription = strings.HasPrefix(strings.ToUpper(descrWord), strings.ToUpper(getWordsFrom(descr)[0])); matchInDescription {
						break
					}
				}
			}

			if matchInDescription {
				productType = tags.ProductType
				if productType == "Для грузовиков и спецтехники" {
					return true
				}
			}
		}
	}

	return false
}
func (pos *position) buildTagsByBrand(brands map[string]avitoCategoriesTagsStruct, priorityDescriptionSource int, posBrand string) (string, string, string, string) {

	var category, goodsType, productType, sparePartType string
	for brand, tags := range brands {
		switch priorityDescriptionSource {
		case 2, 3:
			if pos.AdditionalDescription == "" {
				matchInDescription := caseInsensitiveContains(pos.Description, brand)

				if matchInDescription && len(getWordsFrom(brand)) == 1 {
					words := getWordsFrom(pos.Description)
					for _, word := range words {
						if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
							break
						}
					}
				}

				if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
					category = tags.Category
					goodsType = tags.GoodsType
					productType = tags.ProductType
					sparePartType = tags.SparePartType
					if productType == "Для грузовиков и спецтехники" {
						return category, goodsType, productType, sparePartType
					}
				}
			} else {
				matchInDescription := caseInsensitiveContains(pos.AdditionalDescription+" "+pos.Description, brand)
				if matchInDescription && len(getWordsFrom(brand)) == 1 {
					words := getWordsFrom(pos.AdditionalDescription + " " + pos.Description)
					for _, word := range words {
						if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
							break
						}
					}
				}
				if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
					category = tags.Category
					goodsType = tags.GoodsType
					productType = tags.ProductType
					sparePartType = tags.SparePartType
					if productType == "Для грузовиков и спецтехники" {
						return category, goodsType, productType, sparePartType
					}
				}
			}
		case 5, 7:
			matchInDescription := caseInsensitiveContains(pos.Description+" "+pos.TitleDescription, brand)

			if matchInDescription && len(getWordsFrom(brand)) == 1 {
				words := getWordsFrom(pos.Description + " " + pos.TitleDescription)
				for _, word := range words {
					if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
						break
					}
				}
			}
			if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
				category = tags.Category
				goodsType = tags.GoodsType
				productType = tags.ProductType
				sparePartType = tags.SparePartType
				if productType == "Для грузовиков и спецтехники" {
					return category, goodsType, productType, sparePartType
				}
			}
		default:
			matchInDescription := caseInsensitiveContains(pos.Description, brand)

			if matchInDescription && len(getWordsFrom(brand)) == 1 {
				words := getWordsFrom(pos.Description)
				for _, word := range words {
					if matchInDescription = strings.ToUpper(word) == strings.ToUpper(brand); matchInDescription {
						break
					}
				}
			}
			if matchInDescription || strings.ToUpper(posBrand) == strings.ToUpper(brand) {
				category = tags.Category
				goodsType = tags.GoodsType
				productType = tags.ProductType
				sparePartType = tags.SparePartType
				if productType == "Для грузовиков и спецтехники" {
					return category, goodsType, productType, sparePartType
				}
			}
		}
	}

	return category, goodsType, productType, sparePartType
}
func (pos *position) buildTagsByTruckDescription(truckDescrCategories map[string]avitoCategoriesTagsStruct, priorityDescriptionSource int) (string, string) {

	var sparePartType, technicSparePartType string
	for descr, tags := range truckDescrCategories {
		switch priorityDescriptionSource {
		case 2, 3:

			if pos.AdditionalDescription == "" {
				matchInDescription := caseInsensitiveContains(pos.Description, descr)
				if matchInDescription {
					technicSparePartType = tags.TechnicSparePartType
					sparePartType = tags.SparePartType

					return sparePartType, technicSparePartType
				}
			} else {
				matchInDescription := caseInsensitiveContains(pos.AdditionalDescription+" "+pos.Description, descr)
				if matchInDescription {
					technicSparePartType = tags.TechnicSparePartType
					sparePartType = tags.SparePartType

					return sparePartType, technicSparePartType
				}
			}
		default:
			matchInDescription := caseInsensitiveContains(pos.Description, descr)
			if matchInDescription {
				technicSparePartType = tags.TechnicSparePartType
				sparePartType = tags.SparePartType

				return sparePartType, technicSparePartType
			}
		}
	}

	return sparePartType, technicSparePartType
}

func (pos *position) buildTagsByDescription(avitoCategoriesTags, descrCategories map[string]avitoCategoriesTagsStruct, priorityDescriptionSource int) (string, string, string, string, string, string) {

	var (
		category, goodsType, productType, sparePartType, sparePartType2, descrGoodsGroup string
		matchInDescription                                                               bool
	)
	for descr, tags := range descrCategories {
		switch priorityDescriptionSource {
		case 2, 3:

			if pos.AdditionalDescription == "" {

				matchInDescription = caseInsensitiveContains(pos.Description, descr)
				if matchInDescription && len(getWordsFrom(descr)) == 1 {
					descrWords := getWordsFrom(pos.Description)
					for _, descrWord := range descrWords {
						if matchInDescription = strings.HasPrefix(strings.ToUpper(descrWord), strings.ToUpper(getWordsFrom(descr)[0])); matchInDescription {
							break
						}
					}
				}
				if matchInDescription {
					category = tags.Category
					goodsType = tags.GoodsType
					productType = tags.ProductType
					sparePartType = tags.SparePartType
					sparePartType2 = tags.SparePartType2
					descrGoodsGroup = tags.GoodsGroup
					if category == "" && goodsType == "" && productType == "" && sparePartType == "" && sparePartType2 == "" && tags.GoodsGroup != "" {
						ggID := getGoodsGroupsID(tags.GoodsGroup, tags.GoodsGroup)
						productType = avitoCategoriesTags[ggID].ProductType
						category = avitoCategoriesTags[ggID].Category
						goodsType = avitoCategoriesTags[ggID].GoodsType
						sparePartType = avitoCategoriesTags[ggID].SparePartType
						sparePartType2 = avitoCategoriesTags[ggID].SparePartType2
						descrGoodsGroup = tags.GoodsGroup
					}
					return category, goodsType, productType, sparePartType, descrGoodsGroup, sparePartType2
				}
			} else {
				matchInDescription = caseInsensitiveContains(pos.AdditionalDescription+" "+pos.Description, descr)
				if matchInDescription && len(getWordsFrom(descr)) == 1 {
					descrWords := getWordsFrom(pos.AdditionalDescription + " " + pos.Description)
					for _, descrWord := range descrWords {
						if matchInDescription = strings.HasPrefix(strings.ToUpper(descrWord), strings.ToUpper(getWordsFrom(descr)[0])); matchInDescription {
							break
						}
					}
				}

				if matchInDescription {
					category = tags.Category
					goodsType = tags.GoodsType
					productType = tags.ProductType
					sparePartType = tags.SparePartType
					sparePartType2 = tags.SparePartType2
					descrGoodsGroup = tags.GoodsGroup
					if category == "" && goodsType == "" && productType == "" && sparePartType == "" && sparePartType2 == "" && tags.GoodsGroup != "" {
						ggID := getGoodsGroupsID(tags.GoodsGroup, tags.GoodsGroup)
						productType = avitoCategoriesTags[ggID].ProductType
						category = avitoCategoriesTags[ggID].Category
						goodsType = avitoCategoriesTags[ggID].GoodsType
						sparePartType = avitoCategoriesTags[ggID].SparePartType
						sparePartType2 = avitoCategoriesTags[ggID].SparePartType2
						descrGoodsGroup = tags.GoodsGroup
					}
					return category, goodsType, productType, sparePartType, descrGoodsGroup, sparePartType2
				}
			}
		default:
			matchInDescription = caseInsensitiveContains(pos.Description, descr)

			if matchInDescription && len(getWordsFrom(descr)) == 1 {
				descrWords := getWordsFrom(pos.Description)
				for _, descrWord := range descrWords {
					if matchInDescription = strings.HasPrefix(strings.ToUpper(descrWord), strings.ToUpper(getWordsFrom(descr)[0])); matchInDescription {
						break
					}
				}
			}

			if matchInDescription {
				category = tags.Category
				goodsType = tags.GoodsType
				productType = tags.ProductType
				sparePartType = tags.SparePartType
				sparePartType2 = tags.SparePartType2
				descrGoodsGroup = tags.GoodsGroup
				if category == "" && goodsType == "" && productType == "" && sparePartType == "" && sparePartType2 == "" && tags.GoodsGroup != "" {
					ggID := getGoodsGroupsID(tags.GoodsGroup, tags.GoodsGroup)
					productType = avitoCategoriesTags[ggID].ProductType
					category = avitoCategoriesTags[ggID].Category
					goodsType = avitoCategoriesTags[ggID].GoodsType
					sparePartType = avitoCategoriesTags[ggID].SparePartType
					sparePartType2 = avitoCategoriesTags[ggID].SparePartType2
					descrGoodsGroup = tags.GoodsGroup
				}
				return category, goodsType, productType, sparePartType, descrGoodsGroup, sparePartType2
			}
		}
	}

	return category, goodsType, productType, sparePartType, descrGoodsGroup, sparePartType2
}
