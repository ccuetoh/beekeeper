/*
 * Copyright © 2020 Camilo Hernández <me@camiloh.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package beekeeper

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"time"
	
	"github.com/mitchellh/go-homedir"
)

func getTLSCache() (pemCert []byte , pemKey []byte, err error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return nil, nil, err
	}

	folderPath := filepath.FromSlash(homeDir + "/.beekeeper")
	certPath := filepath.FromSlash(folderPath + "/tls.cert")
	keyPath := filepath.FromSlash(folderPath + "/tls.key")

	if !doesPathExists(certPath) || !doesPathExists(keyPath) {
		return nil, nil, errors.New("not found")
	}

	pemCert, err = ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}

	pemKey, err = ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}

	return pemCert, pemKey, nil
}

func cacheTLS(pemCert []byte , pemKey []byte) (err error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return err
	}

	folderPath := filepath.FromSlash(homeDir + "/.beekeeper")
	certPath := filepath.FromSlash(folderPath + "/tls.cert")
	keyPath := filepath.FromSlash(folderPath + "/tls.key")

	err = createFolderIfNotExist(folderPath)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(certPath, pemCert, 0700)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(keyPath, pemKey, 0700)
	if err != nil {
		return err
	}

	return nil
}

func newSelfSignedCert() (pemCert []byte , pemKey []byte, err error) {
	bits := 4096
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}

	tpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Beekeeper Server"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(2, 0, 0),
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	derCert, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	buf := &bytes.Buffer{}
	err = pem.Encode(buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derCert,
	})
	if err != nil {
		return nil, nil, err
	}

	pemCert = buf.Bytes()

	buf = &bytes.Buffer{}
	err = pem.Encode(buf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return nil, nil, err
	}
	pemKey = buf.Bytes()

	return pemCert, pemKey, nil
}