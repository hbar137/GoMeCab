# GoMeCab

Pure Go implementation of [MeCab](https://taku910.github.io/mecab/), a Japanese morphological analyzer. No CGo, no C dependencies.

Loads standard MeCab compiled dictionaries (IPAdic, UniDic, etc.) directly and produces identical tokenization results.

## Install

```bash
go get gomecab
```

CLI tool:

```bash
go install gomecab/cmd/gomecab@latest
```

## Usage

### Library

```go
package main

import (
    "fmt"
    "gomecab"
)

func main() {
    t, err := gomecab.New("/var/lib/mecab/dic/ipadic-utf8")
    if err != nil {
        panic(err)
    }

    tokens := t.Tokenize("すもももももももものうち")
    for _, tok := range tokens {
        fmt.Printf("%s\t%s\n", tok.Surface, tok.Feature)
    }
}
```

Output:

```
すもも	名詞,一般,*,*,*,*,すもも,スモモ,スモモ
も	助詞,係助詞,*,*,*,*,も,モ,モ
もも	名詞,一般,*,*,*,*,もも,モモ,モモ
も	助詞,係助詞,*,*,*,*,も,モ,モ
もも	名詞,一般,*,*,*,*,もも,モモ,モモ
の	助詞,連体化,*,*,*,*,の,ノ,ノ
うち	名詞,非自立,副詞可能,*,*,*,うち,ウチ,ウチ
```

### CLI

```bash
echo "吾輩は猫である" | gomecab -d /var/lib/mecab/dic/ipadic-utf8
```

## Dictionary

Requires a pre-compiled MeCab dictionary directory containing:

- `sys.dic` — system dictionary
- `unk.dic` — unknown word dictionary
- `matrix.bin` — connection cost matrix
- `char.bin` or `char.def` — character class definitions

On Debian/Ubuntu:

```bash
sudo apt install mecab-ipadic-utf8
```

The dictionary is installed to `/var/lib/mecab/dic/ipadic-utf8`.

## Architecture

- **dict/** — loads MeCab binary dictionaries (double-array trie, token entries, feature strings, connection cost matrix, character classifications)
- **lattice/** — builds the morpheme candidate lattice and runs Viterbi best-path search
- **gomecab.go** — public API (`New`, `Tokenize`)
- **cmd/gomecab/** — command-line tool

## License

MIT

---

# GoMeCab

[MeCab](https://taku910.github.io/mecab/)（日本語形態素解析器）の純粋なGo実装です。CGo不要、C依存なし。

MeCabのコンパイル済み辞書（IPAdic、UniDicなど）をそのまま読み込み、同一の解析結果を出力します。

## インストール

```bash
go get gomecab
```

CLIツール：

```bash
go install gomecab/cmd/gomecab@latest
```

## 使い方

### ライブラリ

```go
package main

import (
    "fmt"
    "gomecab"
)

func main() {
    t, err := gomecab.New("/var/lib/mecab/dic/ipadic-utf8")
    if err != nil {
        panic(err)
    }

    tokens := t.Tokenize("すもももももももものうち")
    for _, tok := range tokens {
        fmt.Printf("%s\t%s\n", tok.Surface, tok.Feature)
    }
}
```

出力：

```
すもも	名詞,一般,*,*,*,*,すもも,スモモ,スモモ
も	助詞,係助詞,*,*,*,*,も,モ,モ
もも	名詞,一般,*,*,*,*,もも,モモ,モモ
も	助詞,係助詞,*,*,*,*,も,モ,モ
もも	名詞,一般,*,*,*,*,もも,モモ,モモ
の	助詞,連体化,*,*,*,*,の,ノ,ノ
うち	名詞,非自立,副詞可能,*,*,*,うち,ウチ,ウチ
```

### コマンドライン

```bash
echo "吾輩は猫である" | gomecab -d /var/lib/mecab/dic/ipadic-utf8
```

## 辞書

以下のファイルを含むMeCabコンパイル済み辞書ディレクトリが必要です：

- `sys.dic` — システム辞書
- `unk.dic` — 未知語辞書
- `matrix.bin` — 連接コスト行列
- `char.bin` または `char.def` — 文字種定義

Debian/Ubuntuの場合：

```bash
sudo apt install mecab-ipadic-utf8
```

辞書は `/var/lib/mecab/dic/ipadic-utf8` にインストールされます。

## 構成

- **dict/** — MeCabバイナリ辞書の読み込み（ダブル配列トライ、トークン、素性文字列、連接コスト行列、文字種分類）
- **lattice/** — 形態素候補ラティスの構築とビタビ最適経路探索
- **gomecab.go** — 公開API（`New`、`Tokenize`）
- **cmd/gomecab/** — コマンドラインツール

## ライセンス

MIT
