package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	mathRand "math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePublicKey(t *testing.T) {
	// функция для очистки файлов с ключами
	removeFile := func(file string) {
		err := os.Remove(file)
		require.NoError(t, err)
	}

	// функция для генерации публичного ключа
	generateFile := func(pathKey, typeKey string) *rsa.PublicKey {
		// Генерация приватного ключа RSA длиной 4096 бит
		privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)

		// Экспорт публичного ключа из приватного ключа
		publicKey := privateKey.PublicKey

		// Сохранение публичного ключа в файл
		pubFile, err := os.Create(pathKey + "public_key.pem")
		require.NoError(t, err)
		defer pubFile.Close()

		pubASN1, err := x509.MarshalPKIXPublicKey(&publicKey)
		require.NoError(t, err)
		err = pem.Encode(pubFile, &pem.Block{
			Type:  typeKey,
			Bytes: pubASN1,
		})
		require.NoError(t, err)
		return &publicKey
	}

	// успешный тест: выполняю парсинг созданного публичного ключа из файла.
	{
		initialKey := generateFile("./", "PUBLIC KEY")
		getKey, err := ParsePublicKey("./" + "public_key.pem")
		require.NoError(t, err)
		// проверяю ключи на равенство
		assert.Equal(t, initialKey.N.Cmp(getKey.N), 0)
		assert.Equal(t, initialKey.E, getKey.E)

		removeFile("./" + "public_key.pem")
	}
	// тест с разными типами ключа
	{
		_ = generateFile("./", "DIFFERENT TYPE KEY")
		_, err := ParsePublicKey("./" + "public_key.pem")
		require.Error(t, err)

		removeFile("./" + "public_key.pem")
	}
	// тест с несуществующим файлом
	{
		_, err := ParsePublicKey("wrong/path/of/file/" + "public_key.pem")
		require.Error(t, err)
	}
	// тест с неверным ключом
	{
		pathKey := "./"

		// Сохранение публичного ключа в файл
		pubFile, err := os.Create(pathKey + "public_key.pem")
		require.NoError(t, err)
		defer pubFile.Close()

		err = pem.Encode(pubFile, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: nil,
		})
		require.NoError(t, err)

		// проверка ключей
		_, err = ParsePublicKey(pathKey + "public_key.pem")
		require.Error(t, err)

		removeFile(pathKey + "public_key.pem")
	}
}

func TestParsePrivateKey(t *testing.T) {
	// функция для очистки файлов с ключами
	removeFile := func(file string) {
		err := os.Remove(file)
		require.NoError(t, err)
	}

	// функция для генерации публичного ключа
	generateFile := func(pathKey, typeKey string) *rsa.PrivateKey {
		// Генерация приватного ключа RSA длиной 4096 бит
		privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)

		// Сохранение приватного ключа в файл
		privFile, err := os.Create(pathKey + "private_key.pem")
		require.NoError(t, err)
		defer privFile.Close()

		err = pem.Encode(privFile, &pem.Block{
			Type:  typeKey,
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
		require.NoError(t, err)
		return privateKey
	}

	// успешный тест: выполняю парсинг созданного публичного ключа из файла.
	{
		initialKey := generateFile("./", "RSA PRIVATE KEY")
		getKey, err := ParsePrivateKey("./" + "private_key.pem")
		require.NoError(t, err)
		// проверяю ключи на равенство
		assert.Equal(t, initialKey.N.Cmp(getKey.N), 0)
		assert.Equal(t, initialKey.E, getKey.E)

		removeFile("./" + "private_key.pem")
	}
	// тест с разными типами ключа
	{
		_ = generateFile("./", "DIFFERENT TYPE KEY")
		_, err := ParsePrivateKey("./" + "private_key.pem")
		require.Error(t, err)

		removeFile("./" + "private_key.pem")
	}
	// тест с несуществующим файлом
	{
		_, err := ParsePrivateKey("wrong/path/of/file/" + "private_key.pem")
		require.Error(t, err)
	}
	// тест с неверным ключом
	{
		pathKey := "./"

		// Сохранение приватного ключа в файл
		privFile, err := os.Create(pathKey + "private_key.pem")
		require.NoError(t, err)
		defer privFile.Close()

		pem.Encode(privFile, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: nil,
		})

		// проверка ключей
		_, err = ParsePrivateKey(pathKey + "private_key.pem")
		require.Error(t, err)

		removeFile("./" + "private_key.pem")
	}
}

func TestGenerateKeys(t *testing.T) {
	// функция для очистки файлов с ключами
	removeFile := func(file string) {
		err := os.Remove(file)
		require.NoError(t, err)
	}

	// проверка файла на существание и содержание данных
	checkFilesExistAndNotEmpty := func(file string) error {
		info, err := os.Stat(file)

		if os.IsNotExist(err) {
			return fmt.Errorf("file %s not exist", file)
		}

		if err != nil {
			return fmt.Errorf("checking file error %s: %v", file, err)
		}

		if info.Size() == 0 {
			return fmt.Errorf("file %s exist, but empty", file)
		}
		return nil
	}

	// successful generating of public and private keys
	{
		pathKeys := "."
		err := GenerateKeys(pathKeys)
		require.NoError(t, err)
		require.NoError(t, checkFilesExistAndNotEmpty(pathKeys+"/private_key.pem"))
		require.NoError(t, checkFilesExistAndNotEmpty(pathKeys+"/public_key.pem"))

		_, err = ParsePrivateKey(pathKeys + "/private_key.pem")
		require.NoError(t, err)

		_, err = ParsePublicKey(pathKeys + "/public_key.pem")
		require.NoError(t, err)

		// очистка тестовых файлов
		removeFile(pathKeys + "/private_key.pem")
		removeFile(pathKeys + "/public_key.pem")
	}
	// wrong path for keys generating
	{
		pathKeys := "wrong path"
		err := GenerateKeys(pathKeys)
		require.Error(t, err)
	}
}

func TestEncryptData(t *testing.T) {
	// функция для очистки файлов с ключами
	removeFile := func(file string) {
		err := os.Remove(file)
		require.NoError(t, err)
	}

	randomData := func(rnd *mathRand.Rand, n int) []byte {
		b := make([]byte, n)
		_, err := rnd.Read(b)
		require.NoError(t, err)
		return b
	}

	decypherData := func(privateKeyPath string, encryptedData []byte) []byte {
		// Парсинг приватного ключа
		privKey, err := ParsePrivateKey(privateKeyPath)
		require.NoError(t, err)

		// Расшифровка данных с использованием RSA с заполнением OAEP (Optimal Asymmetric Encryption Padding)
		decryptedData, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, encryptedData, nil)
		require.NoError(t, err)

		return decryptedData
	}
	// тест с успешной шифровкой данных
	{
		pathKeys := "."
		err := GenerateKeys(pathKeys)
		require.NoError(t, err)

		rnd := mathRand.New(mathRand.NewSource(77))
		data := randomData(rnd, 256)

		// шифровка данных
		encryptData, err := EncryptData(pathKeys+"/public_key.pem", data)
		require.NoError(t, err)

		// расшифровка данных
		decypherData := decypherData(pathKeys+"/private_key.pem", encryptData)
		require.NoError(t, err)

		// проверка равенства изночальных данных и зашифрованных
		assert.Equal(t, data, decypherData)

		// очистка тестовых файлов
		removeFile(pathKeys + "/private_key.pem")
		removeFile(pathKeys + "/public_key.pem")
	}
	// попытка зашифровать данные с ключом по неверному адресу
	{
		rnd := mathRand.New(mathRand.NewSource(79))
		data := randomData(rnd, 256)

		// шифровка данных
		_, err := EncryptData("wrong/path"+"/public_key.pem", data)
		require.Error(t, err)
	}
	// попытка зашифровать пустые данные
	{
		pathKeys := "."
		err := GenerateKeys(pathKeys)
		require.NoError(t, err)

		// шифровка данных
		_, err = EncryptData(pathKeys+"/public_key.pem", nil)
		require.Error(t, err)

		// очистка тестовых файлов
		removeFile(pathKeys + "/private_key.pem")
		removeFile(pathKeys + "/public_key.pem")
	}
	// попытка зашифровать данные неправильным ключом
	{
		rnd := mathRand.New(mathRand.NewSource(83))
		wrongKeyData := randomData(rnd, 256)

		pathKeys := "."

		// Сохранение неправильного публичного ключа в файл
		pubFile, err := os.Create(pathKeys + "/" + "public_key.pem")
		require.NoError(t, err)
		defer pubFile.Close()
		_, err = pubFile.Write(wrongKeyData)
		require.NoError(t, err)

		rnd = mathRand.New(mathRand.NewSource(87))
		data := randomData(rnd, 256)

		// шифровка данных
		_, err = EncryptData(pathKeys+"/public_key.pem", data)
		require.Error(t, err)

		removeFile(pathKeys + "/public_key.pem")
	}
}

func TestDecryptData(t *testing.T) {
	// функция для очистки файлов с ключами
	removeFile := func(file string) {
		err := os.Remove(file)
		require.NoError(t, err)
	}

	randomData := func(rnd *mathRand.Rand, n int) []byte {
		b := make([]byte, n)
		_, err := rnd.Read(b)
		require.NoError(t, err)
		return b
	}

	encryptData := func(publicKeyPath string, data []byte) []byte {
		// Парсинг публичного ключа
		rsaPubKey, err := ParsePublicKey(publicKeyPath)
		require.NoError(t, err)

		// Шифрование данных с использованием RSA с заполнением OAEP (Optimal Asymmetric Encryption Padding)
		encryptedData, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey, data, nil)
		require.NoError(t, err)

		return encryptedData
	}

	// тест с успешной расшифровкой данных
	{
		pathKeys := "."
		err := GenerateKeys(pathKeys)
		require.NoError(t, err)

		rnd := mathRand.New(mathRand.NewSource(91))
		data := randomData(rnd, 256)

		// шифровка данных
		encryptData := encryptData(pathKeys+"/public_key.pem", data)

		// расшифровка данных
		decypherData, err := DecryptData(pathKeys+"/private_key.pem", encryptData)
		require.NoError(t, err)

		// проверка равенства изночальных данных и расшифрованных
		assert.Equal(t, data, decypherData)

		// очистка тестовых файлов
		removeFile(pathKeys + "/private_key.pem")
		removeFile(pathKeys + "/public_key.pem")
	}
	// попытка расшифровать данные с ключом по неверному адресу
	{
		pathKeys := "."
		err := GenerateKeys(pathKeys)
		require.NoError(t, err)

		rnd := mathRand.New(mathRand.NewSource(93))
		data := randomData(rnd, 256)

		// шифровка данных
		encryptData := encryptData(pathKeys+"/public_key.pem", data)

		// попытка расшифровки данных ключом по неверному адресу
		_, err = DecryptData("wrong/path"+"/private_key.pem", encryptData)
		require.Error(t, err)

		// очистка тестовых файлов
		removeFile(pathKeys + "/private_key.pem")
		removeFile(pathKeys + "/public_key.pem")
	}
	// попытка расшифровать пустые данные
	{
		pathKeys := "."
		err := GenerateKeys(pathKeys)
		require.NoError(t, err)

		// расшифровка данных
		_, err = DecryptData(pathKeys+"/private_key.pem", nil)
		require.Error(t, err)

		// очистка тестовых файлов
		removeFile(pathKeys + "/private_key.pem")
		removeFile(pathKeys + "/public_key.pem")
	}
	// попытка расшифровать данные неправильным ключом
	{
		rnd := mathRand.New(mathRand.NewSource(83))
		wrongKeyData := randomData(rnd, 256)

		pathKeys := "."

		// Сохранение неправильного приватного ключа в файл
		pubFile, err := os.Create(pathKeys + "/" + "private_key.pem")
		require.NoError(t, err)
		defer pubFile.Close()
		_, err = pubFile.Write(wrongKeyData)
		require.NoError(t, err)

		rnd = mathRand.New(mathRand.NewSource(87))
		data := randomData(rnd, 256)

		// шифровка данных
		_, err = DecryptData(pathKeys+"/private_key.pem", data)
		require.Error(t, err)

		removeFile(pathKeys + "/private_key.pem")
	}
}
