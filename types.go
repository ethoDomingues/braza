package braza

type Mapper interface {
	ToMap() map[string]any
}

type ManyMapper[C Mapper] []Mapper

func (mm ManyMapper[C]) ToMap() []map[string]any {
	m := []map[string]any{}
	for _, v := range mm {
		m = append(m, v.ToMap())
	}
	return m
}

type Jsonify interface {
	ToJson() any
}

type ManyJsonify[C Jsonify] []C

func (mj ManyJsonify[C]) ToJson() any {
	m := []any{}
	for _, v := range mj {
		m = append(m, v.ToJson())
	}
	return m
}
