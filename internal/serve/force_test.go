package serve

import "testing"

func TestParsePIDsFromSS(t *testing.T) {
	out := `State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
LISTEN 0 4096 127.0.0.1:7710 0.0.0.0:* users:(("tuile",pid=89114,fd=5))`
	pids := parsePIDsFromSS(out)
	if len(pids) != 1 || pids[0] != 89114 {
		t.Fatalf("pids = %v", pids)
	}
}

func TestPortFromListen(t *testing.T) {
	port, err := portFromListen("127.0.0.1:7710")
	if err != nil || port != 7710 {
		t.Fatalf("port = %d err = %v", port, err)
	}
	port, err = portFromListen(":7710")
	if err != nil || port != 7710 {
		t.Fatalf("port = %d err = %v", port, err)
	}
}

func TestUniquePIDs(t *testing.T) {
	got := uniquePIDs([]int{1, 5, 5, 9})
	if len(got) != 2 || got[0] != 5 || got[1] != 9 {
		t.Fatalf("got %v", got)
	}
}
