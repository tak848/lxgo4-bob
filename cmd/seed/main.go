package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/google/uuid"

	infradb "github.com/tak848/lxgo4-bob/internal/infra/db"
	"github.com/tak848/lxgo4-bob/internal/infra/dbgen"
	enums "github.com/tak848/lxgo4-bob/internal/infra/dbgen/dbenums"
	"github.com/tak848/lxgo4-bob/internal/infra/hook"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()

	dsn := os.Getenv("MIGRATE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:5432/taskman?sslmode=disable"
	}

	bobDB, err := infradb.NewDB(ctx, dsn)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer bobDB.Close()

	hook.RegisterHooks()

	type wsData struct {
		name     string
		members  []string
		projects []struct {
			name  string
			tasks []struct {
				title    string
				desc     string
				status   enums.TaskStatus
				priority enums.TaskPriority
				days     int
			}
		}
	}

	workspaces := []wsData{
		{
			name:    "開発チーム",
			members: []string{"田中太郎", "鈴木花子", "佐藤次郎"},
			projects: []struct {
				name  string
				tasks []struct {
					title    string
					desc     string
					status   enums.TaskStatus
					priority enums.TaskPriority
					days     int
				}
			}{
				{
					name: "決済基盤リプレイス",
					tasks: []struct {
						title    string
						desc     string
						status   enums.TaskStatus
						priority enums.TaskPriority
						days     int
					}{
						{"Stripe SDK のバージョンアップ", "v14 → v16 への移行。Breaking changes の調査含む", enums.TaskStatusInProgress, enums.TaskPriorityHigh, 14},
						{"既存テストの修正", "SDK 更新に伴うテストの修正。モック差し替え", enums.TaskStatusTodo, enums.TaskPriorityMedium, 21},
						{"本番切り替え手順書の作成", "ダウンタイムなしの切り替え手順をまとめる", enums.TaskStatusTodo, enums.TaskPriorityLow, 28},
						{"負荷テストの実施", "新 SDK での決済処理のスループット確認", enums.TaskStatusTodo, enums.TaskPriorityHigh, 35},
						{"旧 SDK の削除", "移行完了後に旧コードをクリーンアップ", enums.TaskStatusTodo, enums.TaskPriorityLow, 42},
					},
				},
				{
					name: "管理画面リニューアル",
					tasks: []struct {
						title    string
						desc     string
						status   enums.TaskStatus
						priority enums.TaskPriority
						days     int
					}{
						{"デザインシステムの導入", "shadcn/ui ベースのコンポーネントライブラリ整備", enums.TaskStatusDone, enums.TaskPriorityUrgent, -7},
						{"ユーザー一覧画面の実装", "検索・ソート・ページネーション対応", enums.TaskStatusInProgress, enums.TaskPriorityHigh, 7},
						{"権限管理画面の実装", "ロールベースのアクセス制御 UI", enums.TaskStatusTodo, enums.TaskPriorityMedium, 14},
						{"ダッシュボードの実装", "KPI グラフと直近のアクティビティ表示", enums.TaskStatusInProgress, enums.TaskPriorityMedium, 21},
						{"E2E テストの追加", "Playwright で主要フローをカバー", enums.TaskStatusTodo, enums.TaskPriorityLow, 28},
					},
				},
			},
		},
		{
			name:    "プロダクトチーム",
			members: []string{"山田一郎", "高橋美咲", "渡辺健太"},
			projects: []struct {
				name  string
				tasks []struct {
					title    string
					desc     string
					status   enums.TaskStatus
					priority enums.TaskPriority
					days     int
				}
			}{
				{
					name: "モバイルアプリ v2",
					tasks: []struct {
						title    string
						desc     string
						status   enums.TaskStatus
						priority enums.TaskPriority
						days     int
					}{
						{"プッシュ通知の実装", "Firebase Cloud Messaging 連携", enums.TaskStatusDone, enums.TaskPriorityHigh, -14},
						{"オフラインモードの設計", "ローカル DB とサーバー同期の設計書", enums.TaskStatusInProgress, enums.TaskPriorityUrgent, 7},
						{"画像アップロードの最適化", "リサイズ・圧縮処理のクライアント側実装", enums.TaskStatusTodo, enums.TaskPriorityMedium, 14},
						{"ディープリンク対応", "Universal Links / App Links の設定", enums.TaskStatusTodo, enums.TaskPriorityLow, 21},
						{"ベータテスト配信", "TestFlight / Firebase App Distribution 設定", enums.TaskStatusTodo, enums.TaskPriorityMedium, 28},
					},
				},
				{
					name: "API パフォーマンス改善",
					tasks: []struct {
						title    string
						desc     string
						status   enums.TaskStatus
						priority enums.TaskPriority
						days     int
					}{
						{"スロークエリの特定", "pg_stat_statements で上位 10 件を洗い出し", enums.TaskStatusDone, enums.TaskPriorityUrgent, -21},
						{"インデックス追加", "特定されたスロークエリに対する複合インデックス追加", enums.TaskStatusDone, enums.TaskPriorityHigh, -7},
						{"N+1 クエリの解消", "Preload/ThenLoad を活用した一括取得への書き換え", enums.TaskStatusInProgress, enums.TaskPriorityHigh, 7},
						{"Redis キャッシュの導入", "マスタデータ系のレスポンスキャッシュ", enums.TaskStatusTodo, enums.TaskPriorityMedium, 14},
						{"レスポンスタイム監視の整備", "p95/p99 のアラート設定", enums.TaskStatusTodo, enums.TaskPriorityLow, 21},
					},
				},
			},
		},
	}

	roles := []enums.MemberRole{enums.MemberRoleOwner, enums.MemberRoleEditor, enums.MemberRoleViewer}

	comments := [][]string{
		{"確認しました。LGTM", "対応ありがとうございます！"},
		{"ここ、もう少し詳細な説明が欲しいです", "修正しました。レビューお願いします"},
	}

	slog.Info("seeding workspaces...")
	for _, ws := range workspaces {
		wsID := mustNewV7()
		w, err := dbgen.Workspaces.Insert(&dbgen.WorkspaceSetter{
			ID:   omit.From(wsID),
			Name: omit.From(ws.name),
		}).One(ctx, bobDB)
		if err != nil {
			slog.Error("failed to create workspace", "error", err)
			os.Exit(1)
		}
		slog.Info("created workspace", "id", w.ID, "name", w.Name)

		memberIDs := make([]uuid.UUID, len(ws.members))
		for j, mName := range ws.members {
			mID := mustNewV7()
			memberIDs[j] = mID
			m, err := dbgen.Members.Insert(&dbgen.MemberSetter{
				ID:          omit.From(mID),
				WorkspaceID: omit.From(wsID),
				Name:        omit.From(mName),
				Email:       omit.From(fmt.Sprintf("%s@example.com", mName)),
				Role:        omit.From(roles[j%len(roles)]),
			}).One(ctx, bobDB)
			if err != nil {
				slog.Error("failed to create member", "error", err)
				os.Exit(1)
			}
			slog.Info("created member", "id", m.ID, "name", m.Name)
		}

		for _, proj := range ws.projects {
			pID := mustNewV7()
			p, err := dbgen.Projects.Insert(&dbgen.ProjectSetter{
				ID:          omit.From(pID),
				WorkspaceID: omit.From(wsID),
				Name:        omit.From(proj.name),
				Description: omit.From(fmt.Sprintf("%s のプロジェクト", proj.name)),
				Status:      omit.From(enums.ProjectStatusActive),
			}).One(ctx, bobDB)
			if err != nil {
				slog.Error("failed to create project", "error", err)
				os.Exit(1)
			}
			slog.Info("created project", "id", p.ID, "name", p.Name)

			for k, task := range proj.tasks {
				tID := mustNewV7()
				assigneeID := memberIDs[k%len(memberIDs)]
				dueDate := time.Now().AddDate(0, 0, task.days)
				t, err := dbgen.Tasks.Insert(&dbgen.TaskSetter{
					ID:          omit.From(tID),
					WorkspaceID: omit.From(wsID),
					ProjectID:   omit.From(pID),
					AssigneeID:  omitnull.From(assigneeID),
					Title:       omit.From(task.title),
					Description: omit.From(task.desc),
					Status:      omit.From(task.status),
					Priority:    omit.From(task.priority),
					DueDate:     omitnull.From(dueDate),
				}).One(ctx, bobDB)
				if err != nil {
					slog.Error("failed to create task", "error", err)
					os.Exit(1)
				}
				slog.Info("created task", "id", t.ID, "title", t.Title)

				if k < len(comments) {
					for _, body := range comments[k] {
						cID := mustNewV7()
						c, err := dbgen.TaskComments.Insert(&dbgen.TaskCommentSetter{
							ID:          omit.From(cID),
							WorkspaceID: omit.From(wsID),
							TaskID:      omit.From(tID),
							AuthorID:    omit.From(memberIDs[(k+1)%len(memberIDs)]),
							Body:        omit.From(body),
						}).One(ctx, bobDB)
						if err != nil {
							slog.Error("failed to create comment", "error", err)
							os.Exit(1)
						}
						slog.Info("created comment", "id", c.ID)
					}
				}
			}
		}
	}

	slog.Info("seed completed successfully")
}

func mustNewV7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("uuid.NewV7: %v", err))
	}
	return id
}
