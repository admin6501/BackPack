package node

import "testing"

func TestChangingAddressNeedsNewPasswordBeforeDroppingHostKey(t *testing.T) {
	isolateStore(t)
	if _, err := Add("peer", "192.0.2.1", 22, "root", "old-password"); err != nil {
		t.Fatal(err)
	}
	if err := update(func(s *Store) error {
		s.Nodes[0].Fingerprint = "SHA256:pinned"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetCredentials("peer", "192.0.2.2", 22, "root", ""); err == nil {
		t.Fatal("new unpinned host inherited the old password")
	}
	n, _ := Find("peer")
	if n.Host != "192.0.2.1" || n.Fingerprint != "SHA256:pinned" || n.Password != "old-password" {
		t.Fatalf("rejected edit changed the node: %+v", n)
	}
	if err := SetCredentials("peer", "192.0.2.2", 22, "root", "new-password"); err != nil {
		t.Fatal(err)
	}
	n, _ = Find("peer")
	if n.Host != "192.0.2.2" || n.Fingerprint != "" || n.Password != "new-password" {
		t.Fatalf("accepted edit did not replace the credentials: %+v", n)
	}
}
