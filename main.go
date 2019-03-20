package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	BANK_ZIP  = "http://ykaku.com/ginkokensaku/ginkositen.zip"
	FILE_PATH = "/tmp/ginkositen.zip"
)

var (
	fileCnt         = 0
	banks           = []Bank{}
	bankKanaList    = map[string][]Bank{}
	branches        = map[string][]Branche{}
	brancheKanaList = map[string]map[string][]Branche{}
	kanaMap         = map[string]string{
		"ｱ": "00001",
		"ｲ": "00002",
		"ｳ": "00003",
		"ｴ": "00004",
		"ｵ": "00005",
		"ｶ": "00006",
		"ｷ": "00007",
		"ｸ": "00008",
		"ｹ": "00009",
		"ｺ": "00010",
		"ｻ": "00011",
		"ｼ": "00012",
		"ｽ": "00013",
		"ｾ": "00014",
		"ｿ": "00015",
		"ﾀ": "00016",
		"ﾁ": "00017",
		"ﾂ": "00018",
		"ﾃ": "00019",
		"ﾄ": "00020",
		"ﾅ": "00021",
		"ﾆ": "00022",
		"ﾇ": "00023",
		"ﾈ": "00024",
		"ﾉ": "00025",
		"ﾊ": "00026",
		"ﾋ": "00027",
		"ﾌ": "00028",
		"ﾍ": "00029",
		"ﾎ": "00030",
		"ﾏ": "00031",
		"ﾐ": "00032",
		"ﾑ": "00033",
		"ﾒ": "00034",
		"ﾓ": "00035",
		"ﾔ": "00036",
		"ﾕ": "00037",
		"ﾖ": "00038",
		"ﾗ": "00039",
		"ﾘ": "00040",
		"ﾙ": "00041",
		"ﾚ": "00042",
		"ﾛ": "00043",
		"ﾜ": "00044",
		"ｦ": "00045",
		"ﾝ": "00046",
		"A": "00004",
		"B": "00027",
		"C": "00012",
		"D": "00019",
		"E": "00002",
		"F": "00004",
		"G": "00012",
		"H": "00004",
		"I": "00001",
		"J": "00012",
		"K": "00009",
		"L": "00004",
		"M": "00004",
		"N": "00004",
		"O": "00005",
		"P": "00027",
		"Q": "00007",
		"R": "00001",
		"S": "00004",
		"T": "00019",
		"U": "00037",
		"V": "00028",
		"W": "00016",
		"X": "00004",
		"Y": "00044",
		"Z": "00014",
	}
)

type Bank struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	NameKana string `json:"name_kana"`
}

type Branche struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	NameKana string `json:"name_kana"`
}

func downloadBankData() error {
	res, err := http.Get(BANK_ZIP)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	out, err := os.Create(FILE_PATH)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, res.Body)
	return err
}

func unzip(dest string) error {
	res, err := zip.OpenReader(FILE_PATH)
	if err != nil {
		return err
	}
	defer res.Close()

	for _, f := range res.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		if f.FileInfo().IsDir() {
			path := filepath.Join(dest, f.Name)
			os.MkdirAll(path, f.Mode())
		} else {
			buf := make([]byte, f.UncompressedSize)
			_, err = io.ReadFull(rc, buf)
			if err != nil {
				return err
			}
			path := filepath.Join(dest, f.Name)
			if err = ioutil.WriteFile(path, buf, f.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func readZenginText(txtPath string) ([]string, error) {
	list := []string{}

	f, err := os.Open(txtPath)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(transform.NewReader(f, japanese.ShiftJIS.NewDecoder()))
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func outputJson(dest string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(b)
	fileCnt++

	return nil
}

func resetDataDirectory() error {
	targetDir := []string{
		"./data/bankKana",
		"./data/banks",
		"./data/branchKana",
		"./data/branches",
	}

	for _, path := range targetDir {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		if err := os.Mkdir(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

func trim(str string) string {
	res := strings.Replace(str, " ", "", -1)
	return strings.Replace(res, "\"", "", -1)
}

func parse(list []string) error {
	for _, line := range list {
		arr := strings.Split(line, ",")

		code := arr[0]
		branchCode := arr[1]
		name := trim(arr[3])
		nameKana := trim(arr[2])
		prefixKana := string([]rune(nameKana)[0])

		// 5列目が1なら銀行データ
		if arr[4] == "1" {
			b := Bank{
				Code:     code,
				Name:     name,
				NameKana: nameKana,
			}
			banks = append(banks, b)
			bankKanaList[kanaMap[prefixKana]] = append(bankKanaList[kanaMap[prefixKana]], b)
		} else {
			b := Branche{
				Code:     branchCode,
				Name:     name,
				NameKana: nameKana,
			}
			branches[code] = append(branches[code], b)
			if _, exists := brancheKanaList[code]; !exists {
				brancheKanaList[code] = map[string][]Branche{}
			}
			brancheKanaList[code][kanaMap[prefixKana]] = append(brancheKanaList[code][kanaMap[prefixKana]], b)
		}
	}

	return nil
}

func outputFiles() error {
	// output bank
	for _, v := range banks {
		path := fmt.Sprintf("./data/banks/%s.json", v.Code)
		if err := outputJson(path, v); err != nil {
			return err
		}
	}

	// output bank kana list
	kanaList := []string{}
	for i, v := range bankKanaList {
		path := fmt.Sprintf("./data/bankKana/%s.json", i)
		if err := outputJson(path, v); err != nil {
			return err
		}
		kanaList = append(kanaList, i)
	}
	sort.Strings(kanaList)
	if err := outputJson("./data/bankKana/list.json", kanaList); err != nil {
		return err
	}

	// output bank branches
	for bankCode, list := range branches {
		path := fmt.Sprintf("./data/branches/%s.json", bankCode)
		if err := outputJson(path, list); err != nil {
			return err
		}
	}

	// output bank branch kana list
	for bankCode, list := range brancheKanaList {
		dir := fmt.Sprintf("./data/branchKana/%s", bankCode)
		_, err := os.Stat(dir)
		if err != nil {
			os.Mkdir(dir, 0755)
		}

		kanaList := []string{}
		for i, v := range list {
			path := fmt.Sprintf("%s/%s.json", dir, i)
			if err := outputJson(path, v); err != nil {
				return err
			}
			kanaList = append(kanaList, i)
		}

		sort.Strings(kanaList)
		if err := outputJson(dir+"/list.json", kanaList); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := downloadBankData(); err != nil {
		fmt.Printf("%+v\n", err)
	}

	if err := unzip("/tmp"); err != nil {
		fmt.Printf("%+v\n", err)
	}

	list, err := readZenginText("/tmp/ginkositen.txt")
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	if err := parse(list); err != nil {
		fmt.Printf("%+v\n", err)
	}

	if err := resetDataDirectory(); err != nil {
		fmt.Printf("%+v\n", err)
	}

	err = outputFiles()
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	fmt.Printf("output file count: %d\n", fileCnt)
}
