package database
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// DataStruct 定义了要存储的数据结构
type DataStruct struct {
	DID    string `json:"did"`
	Name string `json:"name"`
	Gender string `json:"gender"`
	Age string `json:"age"`
}

// SaveToFile 将数据保存到 JSON 文件
func SaveToFile(data []DataStruct, filename string) error {
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, file, 0644)
}

// LoadFromFile 从 JSON 文件加载数据
func LoadFromFile(filename string) ([]DataStruct, error) {
	var data []DataStruct
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &data)
	return data, err
}
func FindByDID(data []DataStruct, did string) (*DataStruct, error) {
	for _, d := range data {
		if d.DID == did {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("DID not found")
}

func main() {
	data := []DataStruct{
		{"DID:11111111", "taotao","male","22"},
		{"DID:22222222", "gougou","female","11"},
	}

	filename := "data.json"

	// 保存数据
	if err := SaveToFile(data, filename); err != nil {
		fmt.Println("Error saving data:", err)
		return
	}

	// 加载数据
	loadedData, err := LoadFromFile(filename)
	if err != nil {
		fmt.Println("Error loading data:", err)
		return
	}

	fmt.Println("Loaded Data:", loadedData)
}
