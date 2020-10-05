package vmware

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestComposeFieldProject(t *testing.T) {
	table := []struct {
		pkeyOrig      string
		pkeyAnnotated string
		expected      string
		msg           string
	}{
		{
			msg:           "Все входящие пустые.\nОжидаем на выходе пустоту",
			pkeyOrig:      "",
			pkeyAnnotated: "",
			expected:      "",
		},
		{
			msg:           "В кастоме пробел.\nОжидаем на выходе пустоту",
			pkeyOrig:      " ",
			pkeyAnnotated: "",
			expected:      "",
		},
		{
			msg:           "В кастоме валидный список.\nБерем результат из кастома",
			pkeyOrig:      "AAAA,DDDDD",
			pkeyAnnotated: "",
			expected:      "AAAA,DDDDD",
		},
		{
			msg:           "В кастоме повторы, в аннотации пусто.\nОжидаем на выходе список без повторов",
			pkeyOrig:      "AAAA,AAAA,AAAA",
			pkeyAnnotated: "",
			expected:      "AAAA",
		},
		{
			msg:           "В кастоме повторы, в аннотации валидный список.\nОжидаем мердж без повторов",
			pkeyOrig:      "BBBB,BBBB,BBBB",
			pkeyAnnotated: "CCCC",
			expected:      "BBBB,CCCC",
		},
		{
			msg:           "В кастоме пусто, в аннотации валидный список.\nБерем результат из аннотации",
			pkeyOrig:      "",
			pkeyAnnotated: "DDDD,EEEE",
			expected:      "DDDD,EEEE",
		},
		{
			msg:           "В кастоме повтор с запятыми, в аннотации пусто.\nОжидаем на выходе список без повторов",
			pkeyOrig:      ",FFFF,FFFF,FFFF,FFFF,",
			pkeyAnnotated: "",
			expected:      "FFFF",
		},
		{
			msg:           "Везде повторы.\nОжидаем на выходе список без повторов",
			pkeyOrig:      "GGGG,GGGG,GGGG",
			pkeyAnnotated: "GGGG,GGGG,GGGG",
			expected:      "GGGG",
		},
	}
	for _, tst := range table {
		result := composeFieldProject(tst.pkeyOrig, tst.pkeyAnnotated)
		require.Equal(t, tst.expected, result, tst.msg)
	}
}

func TestFieldExpire(t *testing.T) {
	table := []struct {
		expireOrig      string
		expireAnnotated string
		expected        string
		msg             string
	}{
		{
			msg:             "В кастомном поле пурга какая-то",
			expireOrig:      "01da01.2001",
			expireAnnotated: "",
			expected:        time.Now().AddDate(0, 1, 0).Format("02.01.2006"),
		},
		{
			msg:             "В аннотаци поле пурга какая-то",
			expireOrig:      "",
			expireAnnotated: "фыафыавыфаф",
			expected:        time.Now().AddDate(0, 1, 0).Format("02.01.2006"),
		},
		{
			msg:             "В кастомном старая, но валидная дата.\nОжидается, что дата будет взята из кастомного.",
			expireOrig:      "10.10.2010",
			expireAnnotated: "11.11.2011",
			expected:        "10.10.2010",
		},
		{
			msg:             "В кастомном пусто, а в annotation старая дата.\nДолжны выставить сегодня+месяц.",
			expireOrig:      "",
			expireAnnotated: "01.01.2001",
			expected:        time.Now().AddDate(0, 1, 0).Format("02.01.2006"),
		},
		{
			msg:             "В кастомном старая дата, а в аннотации пусто.\nОставляем дату из кастомного.",
			expireOrig:      "01.01.2001",
			expireAnnotated: "",
			expected:        "01.01.2001",
		},
	}
	for _, tst := range table {
		result := composeFieldExpire(tst.expireOrig, tst.expireAnnotated)
		require.Equal(t, tst.expected, result, tst.msg)
	}
}

func TestDeduplicator(t *testing.T) {
	table := []struct {
		input    []string
		expected []string
	}{
		{
			input:    []string{"AAAA", "BBBB", "CCCC", "", "EEEE", ""},
			expected: []string{"AAAA", "BBBB", "CCCC", "EEEE"},
		},
		{
			input:    []string{"AAAA", "AAAA", "BBBB", "CCCC", "AAAA", "DDDD"},
			expected: []string{"AAAA", "BBBB", "CCCC", "DDDD"},
		},
		{
			input:    []string{"", "", "", "", "AAAAA"},
			expected: []string{"AAAAA"},
		},
		{
			input:    []string{"", "", "", ""},
			expected: nil,
		},
	}
	for _, tst := range table {
		result := deduplicate(tst.input)
		require.Equal(t, tst.expected, result)
	}
}

func TestRebuildAnnotation(t *testing.T) {
	table := []struct {
		input   string
		newNote string
		pkeys   string
		expire  string
		msg     string
	}{
		{
			msg:     "Пустые вводные",
			input:   "",
			newNote: "",
			pkeys:   "",
			expire:  "",
		},
		{
			msg:     "Оба поля и переносы в конце",
			input:   "Владелец: Ушаков\nПроект: ABB\nДо: 01.06.2018\n\n\n\n",
			newNote: "Владелец: Ушаков\n",
			pkeys:   "ABB",
			expire:  "01.06.2018",
		},
		{
			msg:     "Оба поля, список ключей и переносы в конце",
			input:   "Владелец: Ушаков\nПроект: ABB,DFGHFSDG,SADF,ADQAD,FSF\nnadvasdvgasfs:dsvsfs\n\n\n",
			newNote: "Владелец: Ушаков\nnadvasdvgasfs:dsvsfs\n",
			pkeys:   "ABB,DFGHFSDG,SADF,ADQAD,FSF",
			expire:  "",
		},
		{
			msg:     "Оба поля и переносы в конце, дата пустая",
			input:   "Владелец: Ушаков\nПроект: ABB,JGFJG,DXFX\nДо:\n\n\n\n",
			newNote: "Владелец: Ушаков\n",
			pkeys:   "ABB,JGFJG,DXFX",
			expire:  "",
		},
	}
	for _, tst := range table {
		newNote, pkeys, expire := rebuildAnnotation(tst.input)
		require.Equal(t, tst.newNote, newNote, tst.msg)
		require.Equal(t, tst.pkeys, pkeys, tst.msg)
		require.Equal(t, tst.expire, expire, tst.msg)
	}
}
