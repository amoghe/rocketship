package host

import (
	"bytes"
	"io/ioutil"
	"text/template"
	"time"
)

var (
	SudoersFilePath = "/etc/sudoers"
)

func (c *Controller) RewriteSudoersFile() error {
	c.log.Infoln("Rewriting sudoers file")

	contents, err := c.sudoersFileContents()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(SudoersFilePath, contents, 0440)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) sudoersFileContents() ([]byte, error) {
	templateData := struct {
		GenTime string
		Sudoers []string
	}{
		time.Now().String(),
		[]string{"admin"},
	}

	tmpl, err := template.New("sudoers.conf").Parse(sudoersTemplate)
	if err != nil {
		return []byte{}, err
	}

	retbuf := &bytes.Buffer{}
	if err = tmpl.Execute(retbuf, templateData); err != nil {
		return []byte{}, err
	}

	return retbuf.Bytes(), nil
}
