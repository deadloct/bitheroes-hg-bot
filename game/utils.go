package game

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

var DoubleStruckMap = map[rune]rune{
	'A': '𝔸',
	'B': '𝔹',
	'C': 'ℂ',
	'D': '𝔻',
	'E': '𝔼',
	'F': '𝔽',
	'G': '𝔾',
	'H': 'ℍ',
	'I': '𝕀',
	'J': '𝕁',
	'K': '𝕂',
	'L': '𝕃',
	'M': '𝕄',
	'N': 'ℕ',
	'O': '𝕆',
	'P': 'ℙ',
	'Q': 'ℚ',
	'R': 'ℝ',
	'S': '𝕊',
	'T': '𝕋',
	'U': '𝕌',
	'V': '𝕍',
	'W': '𝕎',
	'X': '𝕏',
	'Y': '𝕐',
	'Z': 'ℤ',
	'a': '𝕒',
	'b': '𝕓',
	'c': '𝕔',
	'd': '𝕕',
	'e': '𝕖',
	'f': '𝕗',
	'g': '𝕘',
	'h': '𝕙',
	'i': '𝕚',
	'j': '𝕛',
	'k': '𝕜',
	'l': '𝕝',
	'm': '𝕞',
	'n': '𝕟',
	'o': '𝕠',
	'p': '𝕡',
	'q': '𝕢',
	'r': '𝕣',
	's': '𝕤',
	't': '𝕥',
	'u': '𝕦',
	'v': '𝕧',
	'w': '𝕨',
	'x': '𝕩',
	'y': '𝕪',
	'z': '𝕫',
	'1': '𝟙',
	'2': '𝟚',
	'3': '𝟛',
	'4': '𝟜',
	'5': '𝟝',
	'6': '𝟞',
	'7': '𝟟',
	'8': '𝟠',
	'9': '𝟡',
	'0': '𝟘',
}

func GenerateSequence(n int) []int {
	nums := make([]int, n)
	for i := 0; i < n; i++ {
		nums[i] = i
	}

	return nums
}

func GetRandomInt(min, max int) (int, error) {
	if max-min <= 0 {
		return 0, fmt.Errorf("tried to get random int between [0, %v)", max-min)
	}

	bg := big.NewInt(int64(max - min))
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		return 0, err
	}

	return int(n.Int64()) + min, nil
}

func ToDoubleStruck(str string) string {
	toDS := func(r rune) rune {
		if v, ok := DoubleStruckMap[r]; ok {
			return v
		}

		return r
	}

	return strings.Map(toDS, str)
}
