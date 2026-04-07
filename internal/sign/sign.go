//go:build windows

package sign

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"syscall"
	"unicode"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	X509_ASN_ENCODING              = windows.X509_ASN_ENCODING
	PKCS_7_ASN_ENCODING            = windows.PKCS_7_ASN_ENCODING
	MY_ENCODING_TYPE               = X509_ASN_ENCODING | PKCS_7_ASN_ENCODING
	CERT_STORE_PROV_SYSTEM         = windows.CERT_STORE_PROV_SYSTEM
	CERT_SYSTEM_STORE_CURRENT_USER = windows.CERT_SYSTEM_STORE_CURRENT_USER
	CERT_FIND_SHA1_HASH            = 0x00010000 + 11

	// Флаги для CryptSignMessage
	CMSG_BARE_CONTENT_FLAG           = 0x00000001
	CMSG_DETACHED_FLAG               = 0x00000004
	CRYPT_MESSAGE_SILENT_KEYSET_FLAG = 0x00000040
)

var (
	crypt32 = syscall.NewLazyDLL("crypt32.dll")

	procCryptSignMessage = crypt32.NewProc("CryptSignMessage")
)

// CRYPT_SIGN_MESSAGE_PARA структура для CryptSignMessage
type CRYPT_SIGN_MESSAGE_PARA struct {
	CbSize                  uint32
	MsgEncodingType         uint32
	SigningCert             *windows.CertContext
	HashAlgorithm           CRYPT_ALGORITHM_IDENTIFIER
	PvHashAuxInfo           uintptr
	CMsgCert                uint32
	RgpMsgCert              uintptr
	CMsgCrl                 uint32
	RgpMsgCrl               uintptr
	CAuthAttr               uint32
	RgAuthAttr              uintptr
	CUnauthAttr             uint32
	RgUnauthAttr            uintptr
	Flags                   uint32
	InnerContentType        uint32
	HashEncryptionAlgorithm CRYPT_ALGORITHM_IDENTIFIER
	PvHashEncryptionAuxInfo uintptr
}

type CRYPT_ALGORITHM_IDENTIFIER struct {
	PszObjId   *byte
	Parameters CRYPT_OBJID_BLOB
}

type CRYPT_OBJID_BLOB struct {
	CbData uint32
	PbData *byte
}

func GetCertByThumbprint(thumbprintHex string) (*x509.Certificate, windows.Handle, error) {
	thumbprintHex = removeSpacesAndLower(thumbprintHex)
	tpBytes, err := hex.DecodeString(thumbprintHex)
	if err != nil {
		return nil, 0, errors.New("invalid thumbprint hex string: " + err.Error())
	}

	storeName, err := windows.UTF16PtrFromString("MY")
	if err != nil {
		return nil, 0, err
	}

	hStore, err := windows.CertOpenStore(
		CERT_STORE_PROV_SYSTEM,
		0,
		0,
		CERT_SYSTEM_STORE_CURRENT_USER,
		uintptr(unsafe.Pointer(storeName)),
	)
	if err != nil {
		return nil, 0, err
	}

	var hashBlob windows.CryptHashBlob
	hashBlob.Size = uint32(len(tpBytes))
	hashBlob.Data = &tpBytes[0]

	pCertCtx, err := windows.CertFindCertificateInStore(
		hStore,
		MY_ENCODING_TYPE,
		0,
		CERT_FIND_SHA1_HASH,
		unsafe.Pointer(&hashBlob),
		nil,
	)
	if err != nil {
		windows.CertCloseStore(hStore, 0)
		return nil, 0, err
	}
	if pCertCtx == nil {
		windows.CertCloseStore(hStore, 0)
		return nil, 0, errors.New("certificate with specified thumbprint not found")
	}

	der := (*[1 << 20]byte)(unsafe.Pointer(pCertCtx.EncodedCert))[:pCertCtx.Length:pCertCtx.Length]
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		windows.CertFreeCertificateContext(pCertCtx)
		windows.CertCloseStore(hStore, 0)
		return nil, 0, err
	}

	return cert, hStore, nil
}

// SignJSON создает CMS Signed Data (encapsulated) для JSON данных
func SignJSON(jsonData []byte, thumbprint string) (string, error) {
	// Получаем сертификат и хранилище
	_, hStore, err := GetCertByThumbprint(thumbprint)
	if err != nil {
		return "", fmt.Errorf("failed to get certificate: %w", err)
	}
	defer windows.CertCloseStore(hStore, 0)

	// Находим сертификат снова для получения контекста
	thumbprintHex := removeSpacesAndLower(thumbprint)
	tpBytes, _ := hex.DecodeString(thumbprintHex)

	var hashBlob windows.CryptHashBlob
	hashBlob.Size = uint32(len(tpBytes))
	hashBlob.Data = &tpBytes[0]

	pCertCtx, err := windows.CertFindCertificateInStore(
		hStore,
		MY_ENCODING_TYPE,
		0,
		CERT_FIND_SHA1_HASH,
		unsafe.Pointer(&hashBlob),
		nil,
	)
	if err != nil || pCertCtx == nil {
		return "", errors.New("certificate not found")
	}
	defer windows.CertFreeCertificateContext(pCertCtx)

	// Подготавливаем массив для rgpMsgCert (включаем сертификат подписанта)
	var rgpMsgCert [1]*windows.CertContext
	rgpMsgCert[0] = pCertCtx

	// OID для SHA256: "2.16.840.1.101.3.4.2.1"
	sha256OID := []byte{0x32, 0x2e, 0x31, 0x36, 0x2e, 0x38, 0x34, 0x30, 0x2e, 0x31, 0x2e, 0x31, 0x30, 0x31, 0x2e, 0x33, 0x2e, 0x34, 0x2e, 0x32, 0x2e, 0x31, 0x00}

	// Заполняем параметры подписи
	signPara := CRYPT_SIGN_MESSAGE_PARA{
		CbSize:          uint32(unsafe.Sizeof(CRYPT_SIGN_MESSAGE_PARA{})),
		MsgEncodingType: MY_ENCODING_TYPE,
		SigningCert:     pCertCtx,
		HashAlgorithm: CRYPT_ALGORITHM_IDENTIFIER{
			PszObjId: &sha256OID[0],
			Parameters: CRYPT_OBJID_BLOB{
				CbData: 0,
				PbData: nil,
			},
		},
		CMsgCert:   1,                                       // Количество сертификатов (1 - сертификат подписанта)
		RgpMsgCert: uintptr(unsafe.Pointer(&rgpMsgCert[0])), // Указатель на массив контекстов сертификатов
		Flags:      CRYPT_MESSAGE_SILENT_KEYSET_FLAG,        // Подавление UI-запросов от CSP (опционально, но рекомендуется)
	}

	// Первый вызов - получаем размер
	var signedBlobSize uint32
	pbDataPtr := uintptr(unsafe.Pointer(&jsonData[0]))
	dataSize := uint32(len(jsonData))

	ret, _, err := procCryptSignMessage.Call(
		uintptr(unsafe.Pointer(&signPara)),
		0, // fDetachedSignature = FALSE для encapsulated
		1, // cToBeSigned
		uintptr(unsafe.Pointer(&pbDataPtr)),
		uintptr(unsafe.Pointer(&dataSize)),
		0, // pbSignedBlob = NULL для получения размера
		uintptr(unsafe.Pointer(&signedBlobSize)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to get signature size: %w", err)
	}

	// Второй вызов - получаем саму подпись
	signedBlob := make([]byte, signedBlobSize)
	ret, _, err = procCryptSignMessage.Call(
		uintptr(unsafe.Pointer(&signPara)),
		0, // fDetachedSignature = FALSE для encapsulated
		1, // cToBeSigned
		uintptr(unsafe.Pointer(&pbDataPtr)),
		uintptr(unsafe.Pointer(&dataSize)),
		uintptr(unsafe.Pointer(&signedBlob[0])),
		uintptr(unsafe.Pointer(&signedBlobSize)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to create signature: %w", err)
	}

	// Кодируем в Base64
	return base64.StdEncoding.EncodeToString(signedBlob[:signedBlobSize]), nil
}

func removeSpacesAndLower(thumbprintHex string) string {
	runesHex := []rune{}
	for _, v := range thumbprintHex {
		if unicode.IsSpace(v) {
			continue
		}
		runesHex = append(runesHex, unicode.ToLower(v))
	}
	return string(runesHex)
}
