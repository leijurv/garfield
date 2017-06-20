package main

type Meta struct {
	raw  []byte
	data map[string]interface{}
}

func (meta Meta) GetData(key string) (interface{}, bool) {
	a, b := meta.data[key] //I wish I could do return meta.data[key] but go is stupid =(
	return a, b
}
func (meta Meta) Verify() bool {
	//TODO
	return true
}
