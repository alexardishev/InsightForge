package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"errors"
	"log/slog"
	"strings"
)

func (a *AnalyticsDataCenterService) ListColumnMismatchGroups(ctx context.Context, filter models.ColumnMismatchFilter) ([]models.ColumnMismatchGroup, error) {
	groups, err := a.ColumnMismatchStorage.ListMismatchGroups(ctx, filter)
	if err != nil {
		a.log.Error("ошибка получения групп рассинхрона", slog.String("error", err.Error()))
		return nil, err
	}
	return groups, nil
}

func (a *AnalyticsDataCenterService) GetColumnMismatchGroup(ctx context.Context, id int64) (models.ColumnMismatchGroupWithItems, error) {
	group, err := a.ColumnMismatchStorage.GetMismatchGroup(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrMismatchNotFound) {
			return models.ColumnMismatchGroupWithItems{}, err
		}
		a.log.Error("ошибка получения группы рассинхрона", slog.String("error", err.Error()))
		return models.ColumnMismatchGroupWithItems{}, err
	}

	return group, nil
}

func (a *AnalyticsDataCenterService) ApplyColumnMismatchResolution(ctx context.Context, id int64, resolution models.ColumnMismatchResolution) error {
	const op = "AnalyticsDataCenterService.ApplyColumnMismatchResolution"
	log := a.log.With(slog.String("op", op), slog.Int64("group_id", id))

	group, err := a.ColumnMismatchStorage.GetMismatchGroup(ctx, id)
	if err != nil {
		return err
	}

	view, err := a.SchemaProvider.GetView(ctx, group.Group.SchemaID)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			return err
		}
		log.Error("ошибка получения view", slog.String("error", err.Error()))
		return err
	}

	tableName := strings.ToLower(view.Name)
	viewChanged := false

	for _, decision := range resolution.Renames {
		renameQuery, err := sqlgenerator.GenerateRenameColumnQuery(a.DWHDbName, "public", tableName, decision.OldName, decision.NewName)
		if err != nil {
			log.Error("не удалось подготовить запрос переименования", slog.String("error", err.Error()))
			return err
		}

		if err := a.DWHProvider.RenameColumn(ctx, renameQuery); err != nil {
			log.Error("не удалось переименовать колонку в DWH", slog.String("error", err.Error()))
			return err
		}

		if err := renameColumnInView(&view, models.ColumnRenameSuggestion{
			DatabaseName:  group.Group.DatabaseName,
			SchemaName:    group.Group.SchemaName,
			TableName:     group.Group.TableName,
			OldColumnName: decision.OldName,
			NewColumnName: decision.NewName,
		}); err != nil {
			log.Error("не удалось переименовать колонку в view", slog.String("error", err.Error()))
			return err
		}
		viewChanged = true
	}

	if len(resolution.Deletes) > 0 {
		changed, err := removeColumnsFromView(&view, group.Group.DatabaseName, group.Group.SchemaName, group.Group.TableName, resolution.Deletes)
		if err != nil {
			log.Error("не удалось удалить колонки из view", slog.String("error", err.Error()))
			return err
		}
		viewChanged = viewChanged || changed
	}

	if viewChanged {
		if err := a.SchemaProvider.UpdateView(ctx, view, int(group.Group.SchemaID)); err != nil {
			log.Error("не удалось обновить view", slog.String("error", err.Error()))
			return err
		}
	}

	if err := a.ColumnMismatchStorage.ResolveMismatchGroup(ctx, id); err != nil {
		log.Error("не удалось закрыть группу рассинхронов", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func removeColumnsFromView(view *models.View, database, schema, table string, names []string) (bool, error) {
	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[strings.ToLower(n)] = struct{}{}
	}

	changed := false

	for si := range view.Sources {
		source := &view.Sources[si]
		if source.Name != database {
			continue
		}
		for sci := range source.Schemas {
			sch := &source.Schemas[sci]
			if sch.Name != schema {
				continue
			}
			for ti := range sch.Tables {
				tbl := &sch.Tables[ti]
				if tbl.Name != table {
					continue
				}

				var filtered []models.Column
				for _, col := range tbl.Columns {
					if _, ok := nameSet[strings.ToLower(col.Name)]; ok {
						changed = true
						continue
					}
					filtered = append(filtered, col)
				}
				tbl.Columns = filtered
			}
		}
	}

	return changed, nil
}
