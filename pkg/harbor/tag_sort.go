package harbor

import (
	"strconv"

	"github.com/sirupsen/logrus"
)

type TagsSortByDateDes []*Tag

func (r TagsSortByDateDes) Len() int {
	return len(r)
}

func (r TagsSortByDateDes) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r TagsSortByDateDes) Less(i, j int) bool {
	return r[i].Created.After(r[j].Created)
}

type TagsSortByNameDes []*Tag

func (r TagsSortByNameDes) Len() int {
	return len(r)
}

func (r TagsSortByNameDes) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r TagsSortByNameDes) Less(i, j int) bool {
	v1, err := strconv.Atoi(r[i].Name)
	if err != nil {
		logrus.Errorf("%s it not a number tag, it's dangerous", r[i].Name)
	}
	v2, err := strconv.Atoi(r[j].Name)
	if err != nil {
		logrus.Errorf("%s it not a number, it's dangerous", r[j].Name)
	}
	return v1 > v2
}
