package conf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	data     map[string]interface{}
	rawData  string
	filename string
}

func LoadFile(filename string) *Config {
	c := Config{}
	c.data = make(map[string]interface{})
	c.filename = filename
	if err := c.parse(); err != nil {
		fmt.Println("file parse failed, file err:" + err.Error())
		return nil
	}
	return &c
}

func (c *Config) parse() error {
	f, err := os.Open(c.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	b := new(bytes.Buffer)
	_, err = b.ReadFrom(f)
	if err != nil {
		return err
	}
	c.rawData = b.String()
	err = json.Unmarshal(b.Bytes(), &c.data)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) Get(key string) interface{} {
	val := c.data[key]
	return val
}

//Get data as a int, if not found， return -1
func (c *Config) GetInt(key string) int {
	val, ok := c.data[key]
	if !ok {
		return -1
	}
	return int(val.(float64))
}

func (c *Config) GetIntWithDefault(key string, defaultVal int) int {
	val, ok := c.data[key]
	if !ok {
		return defaultVal
	}
	return int(val.(float64))
}

func (c *Config) GetString(key string) string {
	val, ok := c.data[key]
	if !ok {
		return ""
	}
	return val.(string)
}

func (c *Config) GetStringWithDefault(key string, defaultVal string) string {
	val, ok := c.data[key]
	if !ok {
		return defaultVal
	}
	return val.(string)
}

func (c *Config) GetFloat(key string) float64 {
	val, ok := c.data[key]
	if !ok {
		return -1
	}
	return val.(float64)
}

func (c *Config) GetFloatWithDefault(key string, defaultVal float64) float64 {
	val, ok := c.data[key]
	if !ok {
		return defaultVal
	}
	return val.(float64)
}

func (c *Config) GetBool(key string) bool {
	val, ok := c.data[key]
	if !ok {
		return false
	}
	return val.(bool)
}

func (c *Config) GetBoolWithDefault(key string, defaultVal bool) bool {
	val, ok := c.data[key]
	if !ok {
		return defaultVal
	}
	return val.(bool)
}

func (c *Config) GetJson() string {
	return c.rawData
}
