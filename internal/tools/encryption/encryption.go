package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// Cryptographer - структура хранящая приватный и публичный ключи шифрования с методами для шифровки и расшифровки данных.
type Cryptographer struct {
	publicKeyPath  string
	privateKeyPath string
}

// Initialize инициализирует синглтон структуры шифрования с публичным и приватным ключом.
func Initialize(publicKeyPath, privateKeyPath string) *Cryptographer {
	return &Cryptographer{
		publicKeyPath:  publicKeyPath,
		privateKeyPath: privateKeyPath,
	}
}

// Encrypt шифрует данные, используя публичный ключ.
func (c *Cryptographer) Encrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data for encrypt is error")
	}

	// Парсинг публичного ключа
	rsaPubKey, err := ParsePublicKey(c.publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key error: %w", err)
	}

	// Шифрование данных с использованием RSA с заполнением OAEP (Optimal Asymmetric Encryption Padding)
	encryptedData, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey, data, nil)
	if err != nil {
		return nil, err
	}

	return encryptedData, nil
}

// Decrypt расшифровывает данные, используя приватный ключ.
func (c *Cryptographer) Decrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data for decypher is error")
	}

	// Парсинг приватного ключа
	privKey, err := ParsePrivateKey(c.privateKeyPath)
	if err != nil {
		return nil, err
	}

	// Расшифровка данных с использованием RSA с заполнением OAEP (Optimal Asymmetric Encryption Padding)
	decryptedData, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, data, nil)
	if err != nil {
		return nil, err
	}

	return decryptedData, nil
}

// PublicKeyIsSet - функция для определения того, что задан ли публичный ключ шифрования
func (c *Cryptographer) PublicKeyIsSet() bool {
	return c.publicKeyPath != ""
}

// PrivateKeyIsSet - функция для определения того, что задан ли приватный ключ шифрования
func (c *Cryptographer) PrivateKeyIsSet() bool {
	return c.privateKeyPath != ""
}

// GenerateKeys генерирует и сохраняет пару RSA-ключей
// private_key.pem - приватный ключ, public_key.pem - публичный ключ.
func GenerateKeys(savePath string) error {
	// Генерация приватного ключа RSA длиной 4096 бит
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("generate key error: %w", err)
	}

	// Сохранение приватного ключа в файл
	privFile, err := os.Create(savePath + "/" + "private_key.pem")
	if err != nil {
		return fmt.Errorf("create file to save private key error: %w", err)
	}
	defer privFile.Close()

	err = pem.Encode(privFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return fmt.Errorf("encode value and type of key in file error: %w", err)
	}

	// Экспорт публичного ключа из приватного ключа
	publicKey := privateKey.PublicKey

	// Сохранение публичного ключа в файл
	pubFile, err := os.Create(savePath + "/" + "public_key.pem")
	if err != nil {
		return fmt.Errorf("create file to save pablic key error: %w", err)
	}
	defer pubFile.Close()

	pubASN1, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return err
	}
	err = pem.Encode(pubFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})
	if err != nil {
		return fmt.Errorf("encode value and type of key in file error: %w", err)
	}

	return nil
}

// ParsePublicKey парсит публичный ключ из файла.
func ParsePublicKey(publicKeyFile string) (*rsa.PublicKey, error) {
	// Чтение публичного ключа из файла
	pubKeyData, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return nil, err
	}

	// Декодирование PEM-блока
	block, _ := pem.Decode(pubKeyData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("error of encoding public key")
	}

	// Парсинг публичного ключа
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("wrong type of public key")
	}
	return rsaPubKey, nil
}

// ParsePrivateKey парсит приватный ключ из файла.
func ParsePrivateKey(privateKeyFile string) (*rsa.PrivateKey, error) {
	// Чтение приватного ключа из файла
	privKeyData, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}

	// Декодирование PEM-блока
	block, _ := pem.Decode(privKeyData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("error of encoding private key")
	}

	// Парсинг приватного ключа
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}
