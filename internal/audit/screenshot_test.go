package audit

import (
	"bytes"
	"testing"
)

func TestCameraBytesNeverEnterAuditOrSubscribers(t *testing.T) {
	l, err := Open("", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	l.Frame(1, In, []byte(`{"v":1,"type":"screenshot","seq":4,"width":960,"height":540,"jpeg":"SECRET_IMAGE_BYTES"}`))
	records := l.RecentRecords()
	if len(records) != 1 {
		t.Fatal(records)
	}
	if bytes.Contains(records[0].Frame, []byte("SECRET_IMAGE_BYTES")) || bytes.Contains(records[0].Frame, []byte(`"jpeg"`)) {
		t.Fatal("image leaked", string(records[0].Frame))
	}
	if !bytes.Contains(records[0].Frame, []byte(`"width":960`)) {
		t.Fatal("metadata lost")
	}
}
