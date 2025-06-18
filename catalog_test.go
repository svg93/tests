package main

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type PgStorageStub struct{}

func (*PgStorageStub) ParseProperties(url string) (map[string]Properties, error) {
	return map[string]Properties{
		"BRAND1|NUM1": {
			"goods_group":   "oils",
			"height_mm":     318,
			"viscosity":     "5W-30",
			"liquid_volume": "4.0",
			"acea_spec":     "A3/B4",
			"api_spec":      []any{"SN", "CF"},
			"descr":         "Oil for test",
			"oil_type":      "synthetic",
			"oem_spec":      []any{"First", "Second"},
			"images":        []any{"10603a19f1becb348842b54fd132dd1403.jpeg"},
		},
		"BRAND2|NUM2": {
			"goods_group":   "brake_fluids",
			"dot_spec":      "DOT4",
			"liquid_volume": "1.0",
			"descr":         "Brake for test",
		},
	}, nil
}

func (*PgStorageStub) GetGoodsGroupsID(_ any, posGoodsGroup string) string {

	switch posGoodsGroup {
	case "oils":
		return "1"
	case "brake_fluids":
		return "2"
	default:
		return ""
	}
}

func (*PgStorageStub) GetAvitoTruckDescrCategoriesTags() map[string]AvitoCategoriesTagsStruct {
	return map[string]AvitoCategoriesTagsStruct{}
}

func (*PgStorageStub) GetAvitoDescrCategoriesTags() map[string]AvitoCategoriesTagsStruct {
	return map[string]AvitoCategoriesTagsStruct{}
}

func (*PgStorageStub) GetAvitoBrandCategoriesTags() map[string]AvitoCategoriesTagsStruct {
	return map[string]AvitoCategoriesTagsStruct{}
}

func (*PgStorageStub) GetAvitoCategoriesTags() map[string]AvitoCategoriesTagsStruct {
	return map[string]AvitoCategoriesTagsStruct{
		"17": {
			Category:       "Масла",
			GoodsType:      "Авто масла",
			ProductType:    "Моторное масло",
			SparePartType:  "Двигатель",
			SparePartType2: "Смазочные материалы",
		},
		"2": {
			Category:       "Тормозные жидкости",
			GoodsType:      "Жидкости",
			ProductType:    "Тормозная жидкость",
			SparePartType:  "Тормозная система",
			SparePartType2: "",
		},
	}
}

func (*PgStorageStub) GetAvitoSpec(table, param string) map[string]string {
	return map[string]string{}
}

func (*PgStorageStub) GetAvitoModels() AvitoModelsStruct {
	return AvitoModelsStruct{}
}

func TestProcessXML(t *testing.T) {

	PricegenStorage = new(PgStorageStub)

	type args struct {
		positions      [][]position
		avito          *avitoParams
		params         Params
		goodsGroupPack map[string]map[string]map[string]string
	}

	tests := []struct {
		name       string
		args       args
		wantOffers []xmlOffer
	}{
		{
			name: "Oils",
			args: args{
				positions: [][]position{{
					{
						Brand:          "BRAND1",
						Number:         "NUM1",
						PriceSale:      1500.0,
						GoodsGroupCode: "oils",
						Packing:        "1",
						Description:    "",
					},
				}},
				avito: &avitoParams{
					Address:                      "Test address",
					ManagerName:                  "Manager",
					ContactPhone:                 "+7999000000",
					ContactMethod:                1,
					DateEnd:                      1,
					AdType:                       0,
					ListingFee:                   1,
					AdStatus:                     0,
					Condition:                    0,
					PropertiesURL:                "mock",
					ExcludeOffersWithoutPicture:  false,
					AlwaysGenerateImage:          false,
					DisableAlternativeImage:      false,
					AlternativeImageProxy:        "",
					Availability:                 1,
					TiresQuantityType:            0,
					TiresQuantity:                0,
					AvitoOfferID:                 0,
					VideoURL:                     "",
					CallsDevices:                 []string{"callsDevice1", "callsDevice2"},
					RemoveStatementTypeFromDescr: false,
					AdditionalAddresses:          []string{"AdditionalAddress1", "AdditionalAddress2", "AdditionalAddress1"},
					DisplayAreas:                 []string{"displayArea1", "displayArea2"},
					DeliveryFromPrices: []struct {
						MinPrice      float64  `json:"minPrice"`
						MaxPrice      float64  `json:"maxPrice"`
						DeliveryTypes []string `json:"deliveryTypes"`
					}{
						{MinPrice: 1000, MaxPrice: 2500, DeliveryTypes: []string{"deliveryType1", "deliveryType2"}},
					},
					HidePriceTag: false,
				},
				params: Params{
					PriorityDescriptionSource: 0,
					Localization: Localization{
						WearoutPreOwned:   "Б/у",
						StatementTypeNew:  "Новое",
						ListThBuStatement: "Состояние",
					},
					ImageInc: 1,
				},
				goodsGroupPack: map[string]map[string]map[string]string{
					"oils": {
						"pns":             map[string]string{"viscosity": "SAE", "liquid_volume": "Volume", "acea_spec": "ACEA", "api_spec": "API", "descr": "Description"},
						"translatedprops": map[string]string{"synthetic": "синтетика", "First": "Первая", "Second": ""},
					},
				},
			},
			wantOffers: []xmlOffer{{
				ID:                   "", // Hash, can't check
				Title:                "Oil for test NUM1 BRAND1",
				Brand:                "BRAND1",
				Address:              "Test address",
				ManagerName:          "Manager",
				ContactPhone:         "+7999000000",
				Condition:            "Новое",
				ListingFee:           "Package",
				AdStatus:             "Free",
				ContactMethod:        "По телефону",
				AdType:               "Товар приобретен на продажу",
				ProductType:          "",
				SparePartType:        "",
				TechnicSparePartType: "",
				Category:             "",
				GoodsType:            "",
				Volume:               "4,0 л",
				SAE:                  "5W-30",
				ACEA:                 "A3/B4",
				API:                  apiTagStructSlise{Option: []string{"SN", "CF"}},
				Description: charData{[]byte("<![CDATA[Бренд: BRAND1, артикул: NUM1, Oil for test. <br/><ul>" +
					"<li>SAE: 5W-30.</li> <li>Volume: 4,0.</li> <li>ACEA: A3/B4.</li> <li>API: SN/CF.</li></ul> ]]>")},
				Price:         1500,
				Availability:  "В наличии",
				VendorCode:    "NUM1",
				InternetCalls: "Нет",
				OEMOil:        &[]string{"First", "Second"},
				DisplayAreas:  &[]string{"displayArea1", "displayArea2"},
				Images:        []xmlImage{{URL: "https://pubimg.nodacdn.net/images/10603a419f1be7cb3488842b54f3d132dd14030005.jpeg?1"}},
				Addresses:     &xmlOptions{Options: []xmlOption{{Value: "AdditionalAddress1"}, {Value: "AdditionalAddress2"}}},
				CallsDevices:  &[]string{"callsDevice1", "callsDevice2"},
				Delivery:      &[]string{"deliveryType1", "deliveryType2"},
			}},
		},
		{
			name: "Brake fluids",
			args: args{
				positions: [][]position{{
					{
						Brand:          "BRAND2",
						Number:         "NUM2",
						PriceSale:      500.0,
						GoodsGroupCode: "",
						Packing:        "1",
						Description:    "Description",
					},
				}},
				avito: &avitoParams{
					Address:                       "Test address",
					ManagerName:                   "Manager",
					ContactPhone:                  "+7999000000",
					ContactMethod:                 1,
					DateEnd:                       1,
					AdType:                        0,
					ListingFee:                    1,
					AdStatus:                      0,
					Condition:                     0,
					PropertiesURL:                 "mock",
					ExcludeOffersWithoutPicture:   false,
					AlwaysGenerateImage:           true,
					DisableAlternativeImage:       false,
					AlternativeImageProxy:         "alternativeImageProxy",
					AlternativeImageRequestMethod: "URL-JPG",
					Availability:                  1,
					TiresQuantityType:             0,
					TiresQuantity:                 0,
					AvitoOfferID:                  0,
					VideoURL:                      "https://video.abcp.ru",
					CallsDevices:                  nil,
					RemoveStatementTypeFromDescr:  false,
					AdditionalAddresses:           nil,
					DeliveryFromPrices:            nil,
					HidePriceTag:                  false,
				},
				params: Params{
					PriorityDescriptionSource: 0,
					Localization: Localization{
						WearoutPreOwned:   "Б/у",
						StatementTypeNew:  "Новое",
						ListThBuStatement: "Состояние",
					},
					ImageInc: 1,
				},
				goodsGroupPack: map[string]map[string]map[string]string{
					"brake_fluids": {
						"pns":             map[string]string{"dot_spec": "DOT", "liquid_volume": "Volume", "descr": "Description"},
						"translatedprops": map[string]string{},
					},
				},
			},
			wantOffers: []xmlOffer{{
				ID:            "", // Hash, can't check
				Title:         "Description NUM2 BRAND2",
				Brand:         "BRAND2",
				Address:       "Test address",
				ManagerName:   "Manager",
				ContactPhone:  "+7999000000",
				Condition:     "Новое",
				ListingFee:    "Package",
				AdStatus:      "Free",
				ContactMethod: "По телефону",
				AdType:        "Товар приобретен на продажу",
				ProductType:   "Тормозная жидкость",
				SparePartType: "Тормозная система",
				Category:      "Тормозные жидкости",
				GoodsType:     "Жидкости",
				Volume:        "1,0 л",
				DOT:           "DOT4",
				Description: charData{[]byte("<![CDATA[Бренд: BRAND2, артикул: NUM2, Brake for test. <br/><ul>DOT: DOT4." +
					" Volume: 1,0 л. Description: Brake for test.</li> </ul> ]]>")},
				Price:         500,
				Availability:  "В наличии",
				VendorCode:    "NUM2",
				OEM:           "NUM2",
				InternetCalls: "Нет",
				Images:        []xmlImage{{URL: "alternativeImageProxy/images/brand2/num2/full/1149f97a082eb731bab1e4d0bb281be3e8.jpg?1"}},
				VideoURL:      "https://video.abcp.ru",
			}},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			task := &offersTask{
				id:     i + 1,
				output: make(chan []xmlOffer, 1),
			}

			// Prepare input channel
			posCh := make(chan []position, 1)
			for _, positions := range tt.args.positions {
				posCh <- positions
			}
			close(posCh)

			go task.processXML(posCh, tt.args.avito, tt.args.params, tt.args.goodsGroupPack, nil, nil)
			var got []xmlOffer
			for offers := range task.output {
				got = append(got, offers...)
			}

			if len(got) != len(tt.wantOffers) {
				t.Errorf("[case %v] got %d offers, want %d", tt.name, len(got), len(tt.wantOffers))
				return
			}

			for index := range got {

				gotOffer := got[index]
				wantOffer := tt.wantOffers[index]

				// Ignore ID (hash).
				gotOffer.ID = ""
				wantOffer.ID = ""

				// Ignore DateEnd which depending on current time.
				gotOffer.DateEnd = ""
				wantOffer.DateEnd = ""

				// Only check that Description (which is an XML CDATA) contains the expected substring.
				//if string(wantOffer.Description.Text) != string(charData{}.Text) && !strings.Contains(string(gotOffer.Description.Text), string(wantOffer.Description.Text)) {
				//	t.Errorf("[case %v] got.Description = %q, want substring %q", tt.name, string(gotOffer.Description.Text), string(wantOffer.Description.Text))
				//}

				// Ignore Description because auxiliary parameters adds from map and their order can be changed.
				gotOffer.Description = charData{}
				wantOffer.Description = charData{}

				if diff := cmp.Diff(wantOffer, gotOffer); diff != "" {
					t.Errorf("[case %v] checking offer struct returned unexpected diff (-want +got):\n%s", tt.name, diff)
				}
			}
		})
	}
}

func TestExcludeOfferWithDescriptions(t *testing.T) {

	type args struct {
		includedDesc        []string
		excludedDesc        []string
		description         string
		regexpsIncludedDesc []*regexp.Regexp
		regexpsExcludedDesc []*regexp.Regexp
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Description matches excludedDescriptions, should exclude",
			args: args{
				includedDesc:        nil,
				excludedDesc:        []string{"test product"},
				description:         "This is a test product description",
				regexpsIncludedDesc: nil,
				regexpsExcludedDesc: nil,
			},
			want: true,
		},
		{
			name: "Description matches includedDescriptions, should not exclude",
			args: args{
				includedDesc:        []string{"test product"},
				excludedDesc:        nil,
				description:         "This is a test product description",
				regexpsIncludedDesc: nil,
				regexpsExcludedDesc: nil,
			},
			want: false,
		},
		{
			name: "Description does not match includedDescriptions, should exclude",
			args: args{
				includedDesc:        []string{"something else"},
				excludedDesc:        nil,
				description:         "This is a test product description",
				regexpsIncludedDesc: nil,
				regexpsExcludedDesc: nil,
			},
			want: true,
		},
		{
			name: "Description matches regexpsIncludedDescription, should not exclude",
			args: args{
				includedDesc:        nil,
				excludedDesc:        nil,
				description:         "special match",
				regexpsIncludedDesc: []*regexp.Regexp{regexp.MustCompile(`special match`)},
				regexpsExcludedDesc: nil,
			},
			want: false,
		},
		{
			name: "Description does not match regexpsIncludedDescription, should exclude",
			args: args{
				includedDesc:        nil,
				excludedDesc:        nil,
				description:         "no match here",
				regexpsIncludedDesc: []*regexp.Regexp{regexp.MustCompile(`special match`)},
				regexpsExcludedDesc: nil,
			},
			want: true,
		},
		{
			name: "Description matches regexpsExcludedDescription, should exclude",
			args: args{
				includedDesc:        nil,
				excludedDesc:        nil,
				description:         "do not show this",
				regexpsIncludedDesc: nil,
				regexpsExcludedDesc: []*regexp.Regexp{regexp.MustCompile(`not show`)},
			},
			want: true,
		},
		{
			name: "Description matches both included and excluded (excluded should take precedence)",
			args: args{
				includedDesc:        []string{"show this"},
				excludedDesc:        []string{"do not"},
				description:         "do not show this",
				regexpsIncludedDesc: nil,
				regexpsExcludedDesc: nil,
			},
			want: true,
		},
		{
			name: "Description matches both included and excludedRegex (excluded should take precedence)",
			args: args{
				includedDesc:        []string{"show this"},
				excludedDesc:        nil,
				description:         "description",
				regexpsIncludedDesc: nil,
				regexpsExcludedDesc: []*regexp.Regexp{regexp.MustCompile(`d([a-z]+)on`)},
			},
			want: true,
		},
		{
			name: "Description matches includedRegex and not excluded, should not exclude",
			args: args{
				includedDesc:        nil,
				excludedDesc:        nil,
				description:         "keep this item",
				regexpsIncludedDesc: []*regexp.Regexp{regexp.MustCompile(`keep this`)},
				regexpsExcludedDesc: nil,
			},
			want: false,
		},
		{
			name: "Description matches nothing, all lists empty, should not exclude",
			args: args{
				includedDesc:        nil,
				excludedDesc:        nil,
				description:         "ordinary description",
				regexpsIncludedDesc: nil,
				regexpsExcludedDesc: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := excludeOfferWithDescriptions(tt.args.includedDesc, tt.args.excludedDesc, tt.args.description,
				tt.args.regexpsIncludedDesc, tt.args.regexpsExcludedDesc,
			); got != tt.want {
				t.Errorf("excludeOfferWithDescriptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetXMLParams(t *testing.T) {

	tests := []struct {
		name               string
		props              Properties
		pns                map[string]string
		priorityDescSource int
		want               []xmlParam
	}{
		{
			name: "Simple string property",
			props: Properties{
				"goods_group": "oils",
				"viscosity":   "5W-30",
			},
			pns: map[string]string{
				"viscosity": "Вязкость",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Вязкость", Content: "5W-30"},
			},
		},
		{
			name: "Float property",
			props: Properties{
				"goods_group":   "compressor_oils",
				"liquid_volume": 1.5,
			},
			pns: map[string]string{
				"liquid_volume": "Объем",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Объем", Content: "1.50"},
			},
		},
		{
			name: "Int property",
			props: Properties{
				"goods_group": "batteries",
				"voltage":     12,
			},
			pns: map[string]string{
				"voltage": "Напряжение",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Напряжение", Content: "12"},
			},
		},
		{
			name: "Slice of string property",
			props: Properties{
				"goods_group": "oils",
				"api_spec":    []interface{}{"API SN", "API SM"},
			},
			pns: map[string]string{
				"api_spec": "Спецификации",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Спецификации", Content: "API SN/API SM"},
			},
		},
		{
			name: "String liquid_volume prop",
			props: Properties{
				"goods_group":   "oils",
				"liquid_volume": "1.5",
			},
			pns: map[string]string{
				"api_spec":      "Спецификации",
				"liquid_volume": "Объем",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Объем", Content: "1,5"},
			},
		},
		{
			name: "Unknown property not in pns",
			props: Properties{
				"goods_group": "oils",
				"unknown":     "value",
			},
			pns: map[string]string{
				"viscosity": "Вязкость",
			},
			priorityDescSource: 0,
			want:               []xmlParam{},
		},
		{
			name: "Filter by requiredProps for group",
			props: Properties{
				"goods_group": "bicycles",
				"age":         "adult",
				"extra":       "should not appear",
			},
			pns: map[string]string{
				"age":  "Возраст",
				"type": "Тип",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Возраст", Content: "adult"},
			},
		},
		{
			name: "Wipers connector as slice",
			props: Properties{
				"goods_group": "wipers",
				"connector":   []interface{}{"j-hook (крючок)", "RBTL2.0 (19мм)"},
			},
			pns: map[string]string{
				"connector":    "Коннектор",
				"construction": "Конструкция",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Коннектор", Content: "Hook 9x4"},
			},
		},
		{
			name: "Construction prop",
			props: Properties{
				"goods_group":  "wipers",
				"construction": "большая",
			},
			pns: map[string]string{
				"connector":    "Коннектор",
				"construction": "Конструкция",
			},
			priorityDescSource: 0,
			want: []xmlParam{
				{Name: "Конструкция", Content: "Большая"},
			},
		},
		{
			name: "String property with empty translation",
			props: Properties{
				"goods_group": "oils",
				"viscosity":   "5W-40",
			},
			pns: map[string]string{
				"liquid_volume": "",
			},
			priorityDescSource: 0,
			want:               []xmlParam{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getXMLParams(tt.props, tt.pns, tt.priorityDescSource)
			if !cmp.Equal(got, tt.want) {
				t.Errorf("getXMLParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildTagsByBrand(t *testing.T) {

	brandTags := map[string]AvitoCategoriesTagsStruct{
		"TOYOTA": {
			Category:      "Легковые",
			GoodsType:     "Двигатель",
			ProductType:   "Масла",
			SparePartType: "Фильтры",
		},
		"KAMAZ": {
			Category:      "Грузовики",
			GoodsType:     "Двигатель",
			ProductType:   "Для грузовиков и спецтехники",
			SparePartType: "Трансмиссия",
		},
	}

	tests := []struct {
		name                      string
		pos                       position
		priorityDescriptionSource int
		posBrand                  string
		expectedCategory          string
		expectedGoodsType         string
		expectedProductType       string
		expectedSparePartType     string
	}{
		{
			name: "Brand in Description, match TOYOTA",
			pos: position{
				Description: "Гарантия оригинал TOYOTA",
			},
			priorityDescriptionSource: 0,
			posBrand:                  "",
			expectedCategory:          "Легковые",
			expectedGoodsType:         "Двигатель",
			expectedProductType:       "Масла",
			expectedSparePartType:     "Фильтры",
		},
		{
			name: "Brand in Description, match KAMAZ (truck special case)",
			pos: position{
				Description: "от KAMAZ детали",
			},
			priorityDescriptionSource: 0,
			posBrand:                  "",
			expectedCategory:          "Грузовики",
			expectedGoodsType:         "Двигатель",
			expectedProductType:       "Для грузовиков и спецтехники",
			expectedSparePartType:     "Трансмиссия",
		},
		{
			name: "Brand in posBrand, not in Description",
			pos: position{
				Description: "no brand here",
			},
			priorityDescriptionSource: 0,
			posBrand:                  "KAMAZ",
			expectedCategory:          "Грузовики",
			expectedGoodsType:         "Двигатель",
			expectedProductType:       "Для грузовиков и спецтехники",
			expectedSparePartType:     "Трансмиссия",
		},
		{
			name: "No match found",
			pos: position{
				Description: "no brand here",
			},
			priorityDescriptionSource: 0,
			posBrand:                  "UNKNOWN",
			expectedCategory:          "",
			expectedGoodsType:         "",
			expectedProductType:       "",
			expectedSparePartType:     "",
		},
		{
			name: "priorityDescriptionSource 2 with AdditionalDescription",
			pos: position{
				Description:           "",
				AdditionalDescription: "TOYOTA extra info",
			},
			priorityDescriptionSource: 2,
			posBrand:                  "",
			expectedCategory:          "Легковые",
			expectedGoodsType:         "Двигатель",
			expectedProductType:       "Масла",
			expectedSparePartType:     "Фильтры",
		},
		{
			name: "priorityDescriptionSource 2 without AdditionalDescription",
			pos: position{
				Description:           "Гарантия оригинал TOYOTA",
				AdditionalDescription: "",
			},
			priorityDescriptionSource: 2,
			posBrand:                  "",
			expectedCategory:          "Легковые",
			expectedGoodsType:         "Двигатель",
			expectedProductType:       "Масла",
			expectedSparePartType:     "Фильтры",
		},
		{
			name: "priorityDescriptionSource 5 with TitleDescription",
			pos: position{
				Description:      "",
				TitleDescription: "KAMAZ title",
			},
			priorityDescriptionSource: 5,
			posBrand:                  "",
			expectedCategory:          "Грузовики",
			expectedGoodsType:         "Двигатель",
			expectedProductType:       "Для грузовиков и спецтехники",
			expectedSparePartType:     "Трансмиссия",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, gt, pt, spt := tt.pos.buildTagsByBrand(brandTags, tt.priorityDescriptionSource, tt.posBrand)
			got := []string{cat, gt, pt, spt}
			want := []string{tt.expectedCategory, tt.expectedGoodsType, tt.expectedProductType, tt.expectedSparePartType}
			if !cmp.Equal(got, want) {
				t.Errorf("buildTagsByBrand() got = %v, want %v", got, want)
			}
		})
	}
}

func TestBuildTagsByTruckDescription(t *testing.T) {

	truckDescrCategories := map[string]AvitoCategoriesTagsStruct{
		"коробка передач": {
			SparePartType:        "Трансмиссия",
			TechnicSparePartType: "Детали КПП",
		},
		"двигатель": {
			SparePartType:        "Двигатель",
			TechnicSparePartType: "Головка блока",
		},
	}

	tests := []struct {
		name                      string
		pos                       position
		priorityDescriptionSource int
		wantSparePartType         string
		wantTechnicSparePartType  string
	}{
		{
			name: "match in Description, priority 0",
			pos: position{
				Description: "Ремонт коробка передач ZF",
			},
			priorityDescriptionSource: 0,
			wantSparePartType:         "Трансмиссия",
			wantTechnicSparePartType:  "Детали КПП",
		},
		{
			name: "match in AdditionalDescription, priority 2",
			pos: position{
				Description:           "",
				AdditionalDescription: "двигатель Cummins",
			},
			priorityDescriptionSource: 2,
			wantSparePartType:         "Двигатель",
			wantTechnicSparePartType:  "Головка блока",
		},
		{
			name: "match without AdditionalDescription, priority 2",
			pos: position{
				Description:           "Ремонт коробка передач ZF",
				AdditionalDescription: "",
			},
			priorityDescriptionSource: 2,
			wantSparePartType:         "Трансмиссия",
			wantTechnicSparePartType:  "Детали КПП",
		},
		{
			name: "no match returns empty",
			pos: position{
				Description:           "нет совпадения",
				AdditionalDescription: "",
			},
			priorityDescriptionSource: 0,
			wantSparePartType:         "",
			wantTechnicSparePartType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSpare, gotTechnic := tt.pos.buildTagsByTruckDescription(truckDescrCategories, tt.priorityDescriptionSource)
			if gotSpare != tt.wantSparePartType {
				t.Errorf("buildTagsByTruckDescription() = %q, wantSparePartType %q", gotSpare, tt.wantSparePartType)
			}

			if gotTechnic != tt.wantTechnicSparePartType {
				t.Errorf("buildTagsByTruckDescription() = %q, wantTechnicSparePartType %q", gotTechnic, tt.wantTechnicSparePartType)
			}
		})
	}
}

func TestBuildTagsByDescription(t *testing.T) {

	avitoCategoriesTags := map[string]AvitoCategoriesTagsStruct{
		"1": {
			Category:       "Фильтры",
			GoodsType:      "Масляные фильтры",
			ProductType:    "Масляный фильтр",
			SparePartType:  "Двигатель",
			SparePartType2: "Сменный элемент",
			GoodsGroup:     "oils",
		},
	}

	descrCategories := map[string]AvitoCategoriesTagsStruct{
		"фильтр": {
			Category:       "",
			GoodsType:      "",
			ProductType:    "",
			SparePartType:  "",
			SparePartType2: "",
			GoodsGroup:     "oils",
		},
	}

	PricegenStorage = new(PgStorageStub)

	tests := []struct {
		name                      string
		pos                       position
		priorityDescriptionSource int
		wantCategory              string
		wantGoodsType             string
		wantProductType           string
		wantSparePartType         string
		wantGoodsGroup            string
		wantSparePartType2        string
	}{
		{
			name: "match in Description, with goodsGroupId, priority 0",
			pos: position{
				Description: "фильтр",
			},
			priorityDescriptionSource: 0,
			wantCategory:              "Фильтры",
			wantGoodsType:             "Масляные фильтры",
			wantProductType:           "Масляный фильтр",
			wantSparePartType:         "Двигатель",
			wantGoodsGroup:            "oils",
			wantSparePartType2:        "Сменный элемент",
		},
		{
			name: "match in AdditionalDescription, with goodsGroupId, priority 2",
			pos: position{
				AdditionalDescription: "фильтр",
			},
			priorityDescriptionSource: 2,
			wantCategory:              "Фильтры",
			wantGoodsType:             "Масляные фильтры",
			wantProductType:           "Масляный фильтр",
			wantSparePartType:         "Двигатель",
			wantGoodsGroup:            "oils",
			wantSparePartType2:        "Сменный элемент",
		},
		{
			name: "match without AdditionalDescription, with goodsGroupId, priority 2",
			pos: position{
				Description:           "фильтр",
				AdditionalDescription: "",
			},
			priorityDescriptionSource: 2,
			wantCategory:              "Фильтры",
			wantGoodsType:             "Масляные фильтры",
			wantProductType:           "Масляный фильтр",
			wantSparePartType:         "Двигатель",
			wantGoodsGroup:            "oils",
			wantSparePartType2:        "Сменный элемент",
		},
		{
			name: "no match returns empty",
			pos: position{
				Description: "нет совпадения",
			},
			priorityDescriptionSource: 0,
			wantCategory:              "",
			wantGoodsType:             "",
			wantProductType:           "",
			wantSparePartType:         "",
			wantGoodsGroup:            "",
			wantSparePartType2:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, gType, pType, sType, gGroup, sType2 := tt.pos.buildTagsByDescription(avitoCategoriesTags, descrCategories, tt.priorityDescriptionSource)
			got := []string{cat, gType, pType, sType, gGroup, sType2}
			want := []string{tt.wantCategory, tt.wantGoodsType, tt.wantProductType, tt.wantSparePartType, tt.wantGoodsGroup, tt.wantSparePartType2}
			if !cmp.Equal(got, want) {
				t.Errorf("buildTagsByDescription() = %v, want %v", got, want)
			}
		})
	}
}
