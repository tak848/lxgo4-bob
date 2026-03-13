package handler

import (
	"context"

	"github.com/tak848/lxgo4-bob-playground/internal/oas"
)

func (h *Handler) ListMembers(ctx context.Context, params oas.ListMembersParams) ([]oas.Member, error) {
	list, err := h.Members.List(ctx, params.WsId)
	if err != nil {
		return nil, err
	}
	out := make([]oas.Member, len(list))
	for i, m := range list {
		out[i] = memberToOAS(m)
	}
	return out, nil
}

func (h *Handler) CreateMember(ctx context.Context, req *oas.CreateMemberRequest, params oas.CreateMemberParams) (*oas.Member, error) {
	m, err := h.Members.Create(ctx, params.WsId, req.Name, req.Email, string(req.Role))
	if err != nil {
		return nil, err
	}
	o := memberToOAS(*m)
	return &o, nil
}

func (h *Handler) GetMember(ctx context.Context, params oas.GetMemberParams) (*oas.Member, error) {
	m, err := h.Members.Get(ctx, params.WsId, params.ID)
	if err != nil {
		return nil, err
	}
	o := memberToOAS(*m)
	return &o, nil
}

func (h *Handler) UpdateMember(ctx context.Context, req *oas.UpdateMemberRequest, params oas.UpdateMemberParams) (*oas.Member, error) {
	m, err := h.Members.Update(ctx, params.WsId, params.ID, req.Name, req.Email, string(req.Role))
	if err != nil {
		return nil, err
	}
	o := memberToOAS(*m)
	return &o, nil
}

func (h *Handler) DeleteMember(ctx context.Context, params oas.DeleteMemberParams) error {
	return h.Members.Delete(ctx, params.WsId, params.ID)
}

func memberToOAS(m MemberDTO) oas.Member {
	return oas.Member{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		Email:       m.Email,
		Role:        oas.MemberRole(m.Role),
	}
}
