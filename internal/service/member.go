package service

import (
	"context"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/google/uuid"
	"github.com/stephenafamo/bob"

	"github.com/tak848/lxgo4-bob-playground/internal/handler"
	"github.com/tak848/lxgo4-bob-playground/internal/infra/db"
	"github.com/tak848/lxgo4-bob-playground/internal/infra/dbgen"
	enums "github.com/tak848/lxgo4-bob-playground/internal/infra/dbgen/dbenums"
)

type MemberService struct {
	exec bob.Executor
}

var _ handler.MemberService = (*MemberService)(nil)

func NewMemberService(exec bob.Executor) *MemberService {
	return &MemberService{exec: exec}
}

func (s *MemberService) List(ctx context.Context, wsID uuid.UUID) ([]handler.MemberDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	rows, err := dbgen.Members.Query().All(ctx, exec)
	if err != nil {
		return nil, err
	}
	dtos := make([]handler.MemberDTO, len(rows))
	for i, r := range rows {
		dtos[i] = toMemberDTO(r)
	}
	return dtos, nil
}

func (s *MemberService) Get(ctx context.Context, wsID, id uuid.UUID) (*handler.MemberDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	m, err := dbgen.FindMember(ctx, exec, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	dto := toMemberDTO(m)
	return &dto, nil
}

func (s *MemberService) Create(ctx context.Context, wsID uuid.UUID, name, email, role string) (*handler.MemberDTO, error) {
	_, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("uuid.NewV7: %w", err)
	}
	m, err := dbgen.Members.Insert(&dbgen.MemberSetter{
		ID:          omit.From(id),
		WorkspaceID: omit.From(wsID),
		Name:        omit.From(name),
		Email:       omit.From(email),
		Role:        omit.From(enums.MemberRole(role)),
	}).One(ctx, exec)
	if err != nil {
		return nil, err
	}
	dto := toMemberDTO(m)
	return &dto, nil
}

func (s *MemberService) Update(ctx context.Context, wsID, id uuid.UUID, name, email, role string) (*handler.MemberDTO, error) {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	m, err := dbgen.FindMember(ctx, exec, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if err := m.Update(ctx, exec, &dbgen.MemberSetter{
		Name:  omit.From(name),
		Email: omit.From(email),
		Role:  omit.From(enums.MemberRole(role)),
	}); err != nil {
		return nil, err
	}
	dto := toMemberDTO(m)
	return &dto, nil
}

func (s *MemberService) Delete(ctx context.Context, wsID, id uuid.UUID) error {
	ctx, exec := db.WorkspaceScopedExec(ctx, s.exec, wsID)
	m, err := dbgen.FindMember(ctx, exec, id)
	if err != nil {
		return wrapNotFound(err)
	}
	return m.Delete(ctx, exec)
}

func toMemberDTO(m *dbgen.Member) handler.MemberDTO {
	return handler.MemberDTO{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Name:        m.Name,
		Email:       m.Email,
		Role:        string(m.Role),
	}
}
