package card

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/ebfe/scard"
	"github.com/ubavic/bas-celik/document"
)

type MedicalCard struct {
	smartCard *scard.Card
}

var MEDICAL_ATR = []byte{
	0x3B, 0xF4, 0x13, 0x00, 0x00, 0x81, 0x31, 0xFE,
	0x45, 0x52, 0x46, 0x5a, 0x4F, 0xED,
}

func readMedicalCard(card MedicalCard) (*document.MedicalDocument, error) {
	s1 := []byte{0xF3, 0x81, 0x00, 0x00, 0x02, 0x53, 0x45, 0x52, 0x56, 0x53, 0x5A, 0x4B, 0x01}
	apu, _ := buildAPDU(0x00, 0xA4, 0x04, 0x00, s1, 0)

	_, err := card.smartCard.Transmit(apu)
	if err != nil {
		return nil, err
	}

	doc := document.MedicalDocument{}

	rsp, err := card.readFile([]byte{0x0D, 0x01}, false)
	if err != nil {
		return nil, fmt.Errorf("reading document file: %w", err)
	}

	fields := parseResponse(rsp)
	assignField(fields, 1557, &doc.CardIssueDate)
	document.FormatDate(&doc.CardIssueDate)
	assignField(fields, 1558, &doc.CardExpiryDate)
	document.FormatDate(&doc.CardExpiryDate)
	assignField(fields, 1560, &doc.Language)

	rsp, err = card.readFile([]byte{0x0D, 0x02}, false)
	if err != nil {
		return nil, fmt.Errorf("reading document file: %w", err)
	}

	fields = parseResponse(rsp)
	descramble(fields, 1570)
	assignField(fields, 1570, &doc.SurnameCyrl)
	descramble(fields, 1571)
	assignField(fields, 1571, &doc.Surname)
	descramble(fields, 1572)
	assignField(fields, 1572, &doc.GivenNameCyrl)
	descramble(fields, 1573)
	assignField(fields, 1573, &doc.GivenName)
	assignField(fields, 1574, &doc.DateOfBirth)
	document.FormatDate(&doc.DateOfBirth)
	assignField(fields, 1569, &doc.InsuranceNumber)

	rsp, err = card.readFile([]byte{0x0D, 0x03}, false)
	if err != nil {
		return nil, fmt.Errorf("reading document file: %w", err)
	}

	fields = parseResponse(rsp)
	assignField(fields, 1586, &doc.ValidUntil)
	document.FormatDate(&doc.ValidUntil)
	assignBoolField(fields, 1587, &doc.PermanentlyValid)

	rsp, err = card.readFile([]byte{0x0D, 0x04}, false)
	if err != nil {
		return nil, fmt.Errorf("reading document file: %w", err)
	}

	fields = parseResponse(rsp)
	descramble(fields, 1601)
	assignField(fields, 1601, &doc.ParentNameCyrl)
	descramble(fields, 1602)
	assignField(fields, 1602, &doc.ParentName)
	if string(fields[1603]) == "01" {
		doc.Sex = "Mушко"
	} else {
		doc.Sex = "Женско"
	}
	assignField(fields, 1604, &doc.PersonalNumber)
	descramble(fields, 1605)
	assignField(fields, 1605, &doc.AddressStreet)
	descramble(fields, 1607)
	assignField(fields, 1607, &doc.AddressMunicipality)
	descramble(fields, 1608)
	assignField(fields, 1608, &doc.AddressTown)
	assignField(fields, 1610, &doc.AddressNumber)
	assignField(fields, 1612, &doc.AddressApartmentNumber)
	assignField(fields, 1614, &doc.InsuranceReason)
	descramble(fields, 1615)
	assignField(fields, 1615, &doc.InsuranceDescription)
	descramble(fields, 1616)
	assignField(fields, 1616, &doc.InsuranceHolderRelation)
	assignBoolField(fields, 1617, &doc.InsuranceHolderIsFamilyMember)
	assignField(fields, 1618, &doc.InsuranceHolderPersonalNumber)
	assignField(fields, 1619, &doc.InsuranceHolderInsuranceNumber)
	descramble(fields, 1620)
	assignField(fields, 1620, &doc.InsuranceHolderSurnameCyrl)
	descramble(fields, 1621)
	assignField(fields, 1621, &doc.InsuranceHolderSurname)
	descramble(fields, 1622)
	assignField(fields, 1622, &doc.InsuranceHolderNameCyrl)
	descramble(fields, 1623)
	assignField(fields, 1623, &doc.InsuranceHolderName)
	assignField(fields, 1624, &doc.InsuranceStartDate)
	document.FormatDate(&doc.InsuranceStartDate)
	descramble(fields, 1626)
	assignField(fields, 1626, &doc.AddressState)
	descramble(fields, 1628)
	descramble(fields, 1629)
	descramble(fields, 1630)
	assignField(fields, 1630, &doc.ObligeeName)
	descramble(fields, 1531)
	assignField(fields, 1631, &doc.ObligeePlace)
	assignField(fields, 1632, &doc.ObligeeIdNumber)
	if len(doc.ObligeeIdNumber) == 0 {
		assignField(fields, 1633, &doc.ObligeeIdNumber)
	}
	assignField(fields, 1634, &doc.ObligeeActivity)

	return &doc, nil
}

func descramble(fields map[uint][]byte, tag uint) {
	bs, ok := fields[tag]

	if ok {
		fields[tag] = descrambleBytes(bs)
	} else {
		fields[tag] = []byte{}
	}
}

// never go full retarded with encoding
func descrambleBytes(bs []byte) []byte {
	uperCase := []rune{
		'Ј', 'Љ', 'Њ', 'Ћ', 'Д', 'ђ', 'Е', 'Ж', 'А', 'Б',
		'В', 'Г', 'Д', 'Е', 'Ж', 'З', 'И', 'О', 'К', 'Л',
		'M', 'Н', 'О', 'П', 'Р', 'С', 'Т', 'У', 'Џ', 'Х',
		'Ц', 'Ч', 'Ш',
	}

	lowerCase := []rune{
		'а', 'б', 'в', 'г', 'д', 'е', 'ж', 'з', 'и', 'ђ',
		'к', 'л', 'м', 'н', 'о', 'п', 'р', 'с', 'т', 'у',
		'ф', 'љ', 'ц', 'ч', 'ш', 'х', 'ћ', 'ч', 'џ', 'ф',
	}

	out := make([]byte, 0)

	for i := 0; i < len(bs); i += 2 {
		var toAppend []byte
		if i+1 >= len(bs) {
			break
		} else if bs[i+1] == 0x04 {
			if bs[i] >= 0x08 && bs[i] <= 0x28 {
				toAppend = []byte(string(uperCase[bs[i]-0x08]))
			} else if bs[i] >= 0x30 && bs[i] < 0x4E {
				toAppend = []byte(string(lowerCase[bs[i]-0x30]))
			} else if bs[i] == 0x58 {
				toAppend = []byte("j")
			} else if bs[i] == 0x5A {
				toAppend = []byte("њ")
			} else if bs[i] == 0x5F {
				toAppend = []byte("џ")
			} else {
				println(bs[i])
			}
		} else if bs[i+1] == 0x00 {
			toAppend = []byte{bs[i]}
		} else if bs[i+1] == 0x01 {
			switch bs[i] {
			case 6:
				toAppend = []byte("Ć")
			default:
				toAppend = []byte{}
			}

		} else {
			toAppend = []byte{bs[i], bs[i+1]}
		}
		out = append(out, toAppend...)
	}

	return out
}

func (card MedicalCard) readFile(name []byte, _ bool) ([]byte, error) {
	output := make([]byte, 0)

	_, err := card.selectFile(name)
	if err != nil {
		return nil, fmt.Errorf("selecting file: %w", err)
	}

	data, err := read(card.smartCard, 0, 4)
	if err != nil {
		return nil, fmt.Errorf("reading file header: %w", err)
	}

	offset := uint(len(data))
	if offset < 3 {
		return nil, fmt.Errorf("invalid file header: %w", err)
	}
	length := uint(binary.LittleEndian.Uint16(data[2:]))

	for length > 0 {
		data, err := read(card.smartCard, offset, length)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}

		output = append(output, data...)

		offset += uint(len(data))
		length -= uint(len(data))
	}

	return output, nil
}

func (card MedicalCard) selectFile(name []byte) ([]byte, error) {
	apu, err := buildAPDU(0x00, 0xA4, 0x00, 0x00, name, 0)
	if err != nil {
		return nil, fmt.Errorf("building select apu: %w", err)
	}

	rsp, err := card.smartCard.Transmit(apu)
	if err != nil {
		return nil, fmt.Errorf("selecting file: %w", err)
	}

	return rsp, nil
}

func (card MedicalCard) TestMedicalCard() bool {
	s1 := []byte{0xF3, 0x81, 0x00, 0x00, 0x02, 0x53, 0x45, 0x52, 0x56, 0x53, 0x5A, 0x4B, 0x01}
	apu, _ := buildAPDU(0x00, 0xA4, 0x04, 0x00, s1, 0)

	_, err := card.smartCard.Transmit(apu)
	if err != nil {
		return false
	}

	rsp, err := card.readFile([]byte{0x0D, 0x01}, false)
	if err != nil {
		return false
	}

	fields := parseResponse(rsp)
	descramble(fields, 1553)

	if strings.Compare(string(fields[1553]), "Републички фонд за здравствено осигурање") == 0 {
		return true
	}

	return false
}