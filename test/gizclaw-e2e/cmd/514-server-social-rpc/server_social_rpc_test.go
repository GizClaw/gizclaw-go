package peersocialrpc_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/genx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpcapi"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestServerSocialRPCUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "514-server-social-rpc")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)
	chatroomWorkflow := filepath.Join(h.RepoRoot, "test", "gizclaw-e2e", "setup", "resources", "40-workflow-chatroom.json")
	h.RunCLI("admin", "apply", "-f", chatroomWorkflow, "--context", "admin-a").MustSucceed(t)
	for _, peer := range []string{"peer-a", "peer-b", "peer-c", "peer-d"} {
		h.CreateContext(peer).MustSucceed(t)
		h.RegisterContext(peer, "--sn", peer+"-sn").MustSucceed(t)
	}
	peerB := h.ContextPublicKey("peer-b")
	peerC := h.ContextPublicKey("peer-c")

	assertContactRPCs(t, h)
	assertFriendOTPFailureCases(t, h, peerB)
	requestAB := createAcceptedFriendRequest(t, h, "peer-a", "peer-b", peerB, "123456")
	requestAC := createAcceptedFriendRequest(t, h, "peer-a", "peer-c", peerC, "234567")
	if stringValue(requestAB.WorkspaceName) == "" || stringValue(requestAC.WorkspaceName) == "" {
		t.Fatalf("accepted friend workspaces are empty: ab=%#v ac=%#v", requestAB, requestAC)
	}
	assertFriendPagination(t, h, requestAB, requestAC)
	assertRejectedFriendRequest(t, h, peerB)
	assertChatWorkspaceHistory(t, h, "peer-a", "peer-b", stringValue(requestAB.WorkspaceName), "hello direct chat")

	group := mustRunCLIJSON[rpcapi.FriendGroupCreateResponse](t, h, "connect", "friend-group", "create", "family", "--description", "voice room", "--context", "peer-a")
	if stringValue(group.WorkspaceName) == "" {
		t.Fatalf("friend_group.create workspace_name is empty: %#v", group)
	}
	secondFriendGroup := mustRunCLIJSON[rpcapi.FriendGroupCreateResponse](t, h, "connect", "friend-group", "create", "backup", "--context", "peer-a")
	gotFriendGroup := mustRunCLIJSON[rpcapi.FriendGroupGetResponse](t, h, "connect", "friend-group", "get", stringValue(group.Id), "--context", "peer-a")
	if stringValue(gotFriendGroup.Name) != "family" {
		t.Fatalf("friend_group.get name = %q, want family", stringValue(gotFriendGroup.Name))
	}
	if stringValue(gotFriendGroup.WorkspaceName) != stringValue(group.WorkspaceName) {
		t.Fatalf("friend_group.get workspace_name = %q, want %q", stringValue(gotFriendGroup.WorkspaceName), stringValue(group.WorkspaceName))
	}
	if result := h.RunCLI("connect", "friend-group", "get", stringValue(group.Id), "--context", "peer-d"); result.Err == nil {
		t.Fatal("non-member unexpectedly read group")
	}
	renamedFriendGroup := mustRunCLIJSON[rpcapi.FriendGroupPutResponse](t, h, "connect", "friend-group", "put", stringValue(group.Id), "--name", "family chat", "--context", "peer-a")
	if stringValue(renamedFriendGroup.Name) != "family chat" {
		t.Fatalf("friend_group.put name = %q, want family chat", stringValue(renamedFriendGroup.Name))
	}
	assertFriendGroupPagination(t, h, []string{stringValue(group.Id), stringValue(secondFriendGroup.Id)})

	memberB := mustRunCLIJSON[rpcapi.FriendGroupMemberAddResponse](t, h, "connect", "friend-group", "members", "add", stringValue(group.Id), peerB, "--role", "member", "--context", "peer-a")
	if stringValue(memberB.PeerId) != peerB {
		t.Fatalf("member b peer_id = %q, want %q", stringValue(memberB.PeerId), peerB)
	}
	memberB = mustRunCLIJSON[rpcapi.FriendGroupMemberPutResponse](t, h, "connect", "friend-group", "members", "put", stringValue(group.Id), peerB, "--role", "admin", "--context", "peer-a")
	if memberB.Role == nil || *memberB.Role != rpcapi.FriendGroupMemberRoleAdmin {
		t.Fatalf("member b role = %v, want admin", memberB.Role)
	}
	memberC := mustRunCLIJSON[rpcapi.FriendGroupMemberAddResponse](t, h, "connect", "friend-group", "members", "add", stringValue(group.Id), peerC, "--role", "member", "--context", "peer-b")
	if stringValue(memberC.PeerId) != peerC {
		t.Fatalf("member c peer_id = %q, want %q", stringValue(memberC.PeerId), peerC)
	}
	assertFriendGroupMemberPagination(t, h, stringValue(group.Id))
	assertChatWorkspaceHistory(t, h, "peer-b", "peer-c", stringValue(group.WorkspaceName), "hello group chat")
	assertWorkspaceHistoryDenied(t, h, "peer-d", stringValue(group.WorkspaceName))

	deletedMember := mustRunCLIJSON[rpcapi.FriendGroupMemberDeleteResponse](t, h, "connect", "friend-group", "members", "delete", stringValue(group.Id), peerC, "--context", "peer-b")
	if stringValue(deletedMember.PeerId) != peerC {
		t.Fatalf("friend_group.members.delete peer_id = %q, want %q", stringValue(deletedMember.PeerId), peerC)
	}
	assertWorkspaceHistoryDenied(t, h, "peer-c", stringValue(group.WorkspaceName))
	deletedFriendGroup := mustRunCLIJSON[rpcapi.FriendGroupDeleteResponse](t, h, "connect", "friend-group", "delete", stringValue(secondFriendGroup.Id), "--context", "peer-a")
	if stringValue(deletedFriendGroup.Id) != stringValue(secondFriendGroup.Id) {
		t.Fatalf("friend_group.delete id = %q, want %q", stringValue(deletedFriendGroup.Id), stringValue(secondFriendGroup.Id))
	}
	deletedFriend := mustRunCLIJSON[rpcapi.FriendDeleteResponse](t, h, "connect", "friend", "delete", stringValue(requestAC.Id), "--context", "peer-a")
	if stringValue(deletedFriend.Id) != stringValue(requestAC.Id) {
		t.Fatalf("friend.delete id = %q, want %q", stringValue(deletedFriend.Id), stringValue(requestAC.Id))
	}
	assertWorkspaceHistoryDenied(t, h, "peer-c", stringValue(requestAC.WorkspaceName))
}

func assertContactRPCs(t *testing.T, h *clitest.Harness) {
	t.Helper()

	alice := mustRunCLIJSON[rpcapi.ContactCreateResponse](t, h, "connect", "contact", "create", "--display-name", "Alice", "--phone-number", "+1 555 0100", "--context", "peer-a")
	bob := mustRunCLIJSON[rpcapi.ContactCreateResponse](t, h, "connect", "contact", "create", "--display-name", "Bob", "--phone-number", "+1 555 0101", "--context", "peer-a")
	got := mustRunCLIJSON[rpcapi.ContactGetResponse](t, h, "connect", "contact", "get", stringValue(alice.Id), "--context", "peer-a")
	if stringValue(got.DisplayName) != "Alice" {
		t.Fatalf("contact.get display_name = %q, want Alice", stringValue(got.DisplayName))
	}
	updated := mustRunCLIJSON[rpcapi.ContactPutResponse](t, h, "connect", "contact", "put", stringValue(alice.Id), "--display-name", "Alice Zhang", "--phone-number", "+1 555 0102", "--context", "peer-a")
	if stringValue(updated.DisplayName) != "Alice Zhang" {
		t.Fatalf("contact.put display_name = %q, want Alice Zhang", stringValue(updated.DisplayName))
	}
	if result := h.RunCLI("connect", "contact", "get", stringValue(alice.Id), "--context", "peer-b"); result.Err == nil {
		t.Fatal("peer-b unexpectedly read peer-a contact")
	}
	first := mustRunCLIJSON[rpcapi.ContactListResponse](t, h, "connect", "contact", "list", "--limit", "1", "--context", "peer-a")
	if len(first.Items) != 1 || !first.HasNext || first.NextCursor == nil {
		t.Fatalf("contact first page = %#v, want one item and next cursor", first)
	}
	second := mustRunCLIJSON[rpcapi.ContactListResponse](t, h, "connect", "contact", "list", "--limit", "1", "--cursor", *first.NextCursor, "--context", "peer-a")
	if len(second.Items) != 1 || second.HasNext {
		t.Fatalf("contact second page = %#v, want final item", second)
	}
	deleted := mustRunCLIJSON[rpcapi.ContactDeleteResponse](t, h, "connect", "contact", "delete", stringValue(bob.Id), "--context", "peer-a")
	if stringValue(deleted.Id) != stringValue(bob.Id) {
		t.Fatalf("contact.delete id = %q, want %q", stringValue(deleted.Id), stringValue(bob.Id))
	}
}

func createAcceptedFriendRequest(t *testing.T, h *clitest.Harness, fromContext, toContext, toPeerID, code string) rpcapi.FriendObject {
	t.Helper()

	mustRunCLIJSON[rpcapi.ServerGetRunStatusResponse](t, h, "connect", "run-status", "--friend-otp", code, "--context", toContext)
	bad := h.RunCLI("connect", "friend", "requests", "create", toPeerID, "--code", "000000", "--context", fromContext)
	if bad.Err == nil {
		t.Fatal("friend request with wrong device-reported OTP unexpectedly succeeded")
	}
	mustRunCLIJSON[rpcapi.ServerGetRunStatusResponse](t, h, "connect", "run-status", "--friend-otp", code, "--context", toContext)
	req := mustRunCLIJSON[rpcapi.FriendRequestCreateResponse](t, h, "connect", "friend", "requests", "create", toPeerID, "--code", code, "--message", "hi", "--context", fromContext)
	if req.State == nil || *req.State != rpcapi.FriendRequestStatePending {
		t.Fatalf("friend request state = %v, want pending", req.State)
	}
	incoming := mustRunCLIJSON[rpcapi.FriendRequestListResponse](t, h, "connect", "friend", "requests", "list", "--box", "incoming", "--state", "pending", "--limit", "1", "--context", toContext)
	if len(incoming.Items) != 1 || stringValue(incoming.Items[0].Id) != stringValue(req.Id) {
		t.Fatalf("incoming friend requests = %#v, want %q", incoming, stringValue(req.Id))
	}
	accepted := mustRunCLIJSON[rpcapi.FriendRequestAcceptResponse](t, h, "connect", "friend", "requests", "accept", stringValue(req.Id), "--context", toContext)
	if accepted.State == nil || *accepted.State != rpcapi.FriendRequestStateAccepted {
		t.Fatalf("accepted friend request state = %v, want accepted", accepted.State)
	}
	acceptedAgain := mustRunCLIJSON[rpcapi.FriendRequestAcceptResponse](t, h, "connect", "friend", "requests", "accept", stringValue(req.Id), "--context", toContext)
	if stringValue(acceptedAgain.Id) != stringValue(req.Id) || acceptedAgain.State == nil || *acceptedAgain.State != rpcapi.FriendRequestStateAccepted {
		t.Fatalf("second accept = %#v, want same accepted request", acceptedAgain)
	}
	friends := mustRunCLIJSON[rpcapi.FriendListResponse](t, h, "connect", "friend", "list", "--context", fromContext)
	for _, friend := range friends.Items {
		if stringValue(friend.PeerId) == toPeerID {
			return friend
		}
	}
	t.Fatalf("friend relation with %s not found in %#v", toPeerID, friends)
	return rpcapi.FriendObject{}
}

func assertFriendOTPFailureCases(t *testing.T, h *clitest.Harness, peerB string) {
	t.Helper()

	if result := h.RunCLI("connect", "friend", "requests", "create", peerB, "--context", "peer-a"); result.Err == nil {
		t.Fatal("friend request without code unexpectedly succeeded")
	}
	if result := h.RunCLI("connect", "run-status", "--friend-otp", "abc123", "--context", "peer-b"); result.Err == nil {
		t.Fatal("malformed device friend OTP unexpectedly reported")
	}
	if result := h.RunCLI("connect", "friend", "requests", "create", peerB, "--code", "abc123", "--context", "peer-a"); result.Err == nil {
		t.Fatal("friend request with malformed code unexpectedly succeeded")
	}
	mustRunCLIJSON[rpcapi.ServerGetRunStatusResponse](t, h, "connect", "run-status", "--friend-otp", "456789", "--context", "peer-b")
	time.Sleep(3 * time.Second)
	if result := h.RunCLI("connect", "friend", "requests", "create", peerB, "--code", "456789", "--context", "peer-a"); result.Err == nil {
		t.Fatal("friend request with expired code unexpectedly succeeded")
	}

	mustRunCLIJSON[rpcapi.ServerGetRunStatusResponse](t, h, "connect", "run-status", "--friend-otp", "567890", "--context", "peer-b")
	req := mustRunCLIJSON[rpcapi.FriendRequestCreateResponse](t, h, "connect", "friend", "requests", "create", peerB, "--code", "567890", "--context", "peer-c")
	if result := h.RunCLI("connect", "friend", "requests", "create", peerB, "--code", "567890", "--context", "peer-a"); result.Err == nil {
		t.Fatal("friend request with already-consumed code unexpectedly succeeded")
	}
	rejected := mustRunCLIJSON[rpcapi.FriendRequestRejectResponse](t, h, "connect", "friend", "requests", "reject", stringValue(req.Id), "--context", "peer-b")
	if rejected.State == nil || *rejected.State != rpcapi.FriendRequestStateRejected {
		t.Fatalf("rejected consumed-code setup request state = %v, want rejected", rejected.State)
	}
}

func assertRejectedFriendRequest(t *testing.T, h *clitest.Harness, peerB string) {
	t.Helper()

	mustRunCLIJSON[rpcapi.ServerGetRunStatusResponse](t, h, "connect", "run-status", "--friend-otp", "345678", "--context", "peer-b")
	req := mustRunCLIJSON[rpcapi.FriendRequestCreateResponse](t, h, "connect", "friend", "requests", "create", peerB, "--code", "345678", "--context", "peer-c")
	rejected := mustRunCLIJSON[rpcapi.FriendRequestRejectResponse](t, h, "connect", "friend", "requests", "reject", stringValue(req.Id), "--context", "peer-b")
	if rejected.State == nil || *rejected.State != rpcapi.FriendRequestStateRejected {
		t.Fatalf("rejected friend request state = %v, want rejected", rejected.State)
	}
}

func assertFriendPagination(t *testing.T, h *clitest.Harness, firstFriend, secondFriend rpcapi.FriendObject) {
	t.Helper()

	first := mustRunCLIJSON[rpcapi.FriendListResponse](t, h, "connect", "friend", "list", "--limit", "1", "--context", "peer-a")
	if len(first.Items) != 1 || !first.HasNext || first.NextCursor == nil {
		t.Fatalf("friend first page = %#v, want one item and next cursor", first)
	}
	second := mustRunCLIJSON[rpcapi.FriendListResponse](t, h, "connect", "friend", "list", "--limit", "1", "--cursor", *first.NextCursor, "--context", "peer-a")
	if len(second.Items) != 1 || second.HasNext {
		t.Fatalf("friend second page = %#v, want final item", second)
	}
	got := map[string]bool{stringValue(first.Items[0].Id): true, stringValue(second.Items[0].Id): true}
	if !got[stringValue(firstFriend.Id)] || !got[stringValue(secondFriend.Id)] {
		t.Fatalf("friend pagination ids = %#v, want %q and %q", got, stringValue(firstFriend.Id), stringValue(secondFriend.Id))
	}
	requests := mustRunCLIJSON[rpcapi.FriendRequestListResponse](t, h, "connect", "friend", "requests", "list", "--box", "outgoing", "--limit", "1", "--context", "peer-a")
	if len(requests.Items) != 1 || !requests.HasNext || requests.NextCursor == nil {
		t.Fatalf("friend request first page = %#v, want pagination", requests)
	}
	requests = mustRunCLIJSON[rpcapi.FriendRequestListResponse](t, h, "connect", "friend", "requests", "list", "--box", "outgoing", "--limit", "1", "--cursor", *requests.NextCursor, "--context", "peer-a")
	if len(requests.Items) != 1 || requests.HasNext {
		t.Fatalf("friend request second page = %#v, want final item", requests)
	}
}

func assertFriendGroupPagination(t *testing.T, h *clitest.Harness, wantIDs []string) {
	t.Helper()

	first := mustRunCLIJSON[rpcapi.FriendGroupListResponse](t, h, "connect", "friend-group", "list", "--limit", "1", "--context", "peer-a")
	if len(first.Items) != 1 || !first.HasNext || first.NextCursor == nil {
		t.Fatalf("group first page = %#v, want one item and next cursor", first)
	}
	second := mustRunCLIJSON[rpcapi.FriendGroupListResponse](t, h, "connect", "friend-group", "list", "--limit", "1", "--cursor", *first.NextCursor, "--context", "peer-a")
	if len(second.Items) != 1 || second.HasNext {
		t.Fatalf("group second page = %#v, want final item", second)
	}
	got := map[string]bool{stringValue(first.Items[0].Id): true, stringValue(second.Items[0].Id): true}
	for _, id := range wantIDs {
		if !got[id] {
			t.Fatalf("group pagination ids = %#v, missing %q", got, id)
		}
	}
}

func assertFriendGroupMemberPagination(t *testing.T, h *clitest.Harness, friendGroupID string) {
	t.Helper()

	first := mustRunCLIJSON[rpcapi.FriendGroupMemberListResponse](t, h, "connect", "friend-group", "members", "list", friendGroupID, "--limit", "1", "--context", "peer-a")
	if len(first.Items) != 1 || !first.HasNext || first.NextCursor == nil {
		t.Fatalf("friend group member first page = %#v, want one item and next cursor", first)
	}
	second := mustRunCLIJSON[rpcapi.FriendGroupMemberListResponse](t, h, "connect", "friend-group", "members", "list", friendGroupID, "--limit", "1", "--cursor", *first.NextCursor, "--context", "peer-a")
	if len(second.Items) != 1 {
		t.Fatalf("friend group member second page = %#v, want one item", second)
	}
}

func mustRunCLIJSON[T any](t *testing.T, h *clitest.Harness, args ...string) T {
	t.Helper()

	result := h.RunCLI(args...)
	result.MustSucceed(t)
	var out T
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		t.Fatalf("decode %q JSON: %v\nstdout:\n%s\nstderr:\n%s", args, err, result.Stdout, result.Stderr)
	}
	return out
}

func assertChatWorkspaceHistory(t *testing.T, h *clitest.Harness, writerContext, readerContext, workspaceName, text string) {
	t.Helper()

	writer := h.ConnectClientFromContext(writerContext)
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := writer.SetServerRunWorkspace(ctx, "social.chat.workspace.set", rpcapi.ServerSetRunWorkspaceRequest{WorkspaceName: workspaceName}); err != nil {
		t.Fatalf("%s set run workspace %q: %v", writerContext, workspaceName, err)
	}
	state, err := writer.ReloadServerRunWorkspace(ctx, "social.chat.workspace.reload")
	if err != nil {
		t.Fatalf("%s reload run workspace %q: %v", writerContext, workspaceName, err)
	}
	if state.RuntimeState != rpcapi.PeerRunStatusStateRunning {
		t.Fatalf("%s reload workspace state = %#v", writerContext, state)
	}
	out, err := writer.Transform(ctx, "chatroom", chatTextStream(text))
	if err != nil {
		t.Fatalf("%s transform chat text: %v", writerContext, err)
	}
	defer out.Close()

	reader := h.ConnectClientFromContext(readerContext)
	defer reader.Close()
	entry := waitForWorkspaceHistoryText(t, ctx, reader, workspaceName, text)
	got, err := reader.GetWorkspaceHistory(ctx, "social.chat.history.get", rpcapi.WorkspaceHistoryGetRequest{
		WorkspaceName: workspaceName,
		HistoryId:     entry.Id,
	})
	if err != nil {
		t.Fatalf("%s workspace history get %q: %v", readerContext, entry.Id, err)
	}
	if got.Text != text || got.Type != rpcapi.PeerRunHistoryEntryTypeGear || got.GearId == nil || *got.GearId != h.ContextPublicKey(writerContext) {
		t.Fatalf("workspace history get = %#v, want text %q from %s", got, text, writerContext)
	}
	play, err := writer.PlayServerRunWorkspaceHistory(ctx, "social.chat.history.play", rpcapi.ServerPlayRunWorkspaceHistoryRequest{HistoryId: entry.Id})
	if err != nil {
		t.Fatalf("%s workspace history play %q: %v", writerContext, entry.Id, err)
	}
	if !play.Accepted {
		t.Fatalf("workspace history play = %#v, want accepted", play)
	}
}

func assertWorkspaceHistoryDenied(t *testing.T, h *clitest.Harness, contextName, workspaceName string) {
	t.Helper()

	client := h.ConnectClientFromContext(contextName)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.ListWorkspaceHistory(ctx, "social.chat.history.denied", rpcapi.WorkspaceHistoryListRequest{WorkspaceName: workspaceName}); err == nil {
		t.Fatalf("%s unexpectedly listed workspace history for %q", contextName, workspaceName)
	}
}

func waitForWorkspaceHistoryText(t *testing.T, ctx context.Context, client interface {
	ListWorkspaceHistory(context.Context, string, rpcapi.WorkspaceHistoryListRequest) (*rpcapi.WorkspaceHistoryListResponse, error)
}, workspaceName, text string) rpcapi.PeerRunHistoryEntry {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for {
		list, err := client.ListWorkspaceHistory(ctx, "social.chat.history.list", rpcapi.WorkspaceHistoryListRequest{WorkspaceName: workspaceName})
		if err == nil {
			for _, item := range list.Items {
				if item.Text == text {
					return item
				}
			}
			lastErr = nil
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			t.Fatalf("history text %q not found in workspace %q, last error: %v", text, workspaceName, lastErr)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func chatTextStream(text string) genx.Stream {
	return &sliceStream{chunks: []*genx.MessageChunk{
		{Role: genx.RoleUser, Name: "transcript", Part: genx.Text(text), Ctrl: &genx.StreamCtrl{StreamID: "chat-text", Label: "transcript"}},
		{Role: genx.RoleUser, Name: "transcript", Part: genx.Text(""), Ctrl: &genx.StreamCtrl{StreamID: "chat-text", Label: "transcript", EndOfStream: true}},
	}}
}

type sliceStream struct {
	chunks []*genx.MessageChunk
}

func (s *sliceStream) Next() (*genx.MessageChunk, error) {
	if len(s.chunks) == 0 {
		return nil, genx.ErrDone
	}
	chunk := s.chunks[0]
	s.chunks = s.chunks[1:]
	return chunk, nil
}

func (s *sliceStream) Close() error {
	s.chunks = nil
	return nil
}

func (s *sliceStream) CloseWithError(error) error {
	s.chunks = nil
	return nil
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
