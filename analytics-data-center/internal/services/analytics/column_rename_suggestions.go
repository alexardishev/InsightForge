package serviceanalytics

import (
	"analyticDataCenter/analytics-data-center/internal/domain/models"
	sqlgenerator "analyticDataCenter/analytics-data-center/internal/lib/SQLGenerator"
	"analyticDataCenter/analytics-data-center/internal/storage"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

func (a *AnalyticsDataCenterService) AcceptColumnRenameSuggestion(ctx context.Context, id int64) error {
	const op = "AnalyticsDataCenterService.AcceptColumnRenameSuggestion"
	log := a.log.With(slog.String("op", op), slog.Int64("suggestion_id", id))

	suggestion, err := a.RenameSuggestionStorage.GetSuggestionByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrSuggestionNotFound) {
			return err
		}
		log.Error("failed to get suggestion", slog.String("error", err.Error()))
		return err
	}

	view, err := a.SchemaProvider.GetView(ctx, suggestion.SchemaID)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			log.Warn("view not found for suggestion", slog.String("error", err.Error()))
			return err
		}
		log.Error("failed to get view", slog.String("error", err.Error()))
		return err
	}

	tableName := view.Name
	renameQuery, err := sqlgenerator.GenerateRenameColumnQuery(a.DWHDbName, "public", tableName, suggestion.OldColumnName, suggestion.NewColumnName)
	if err != nil {
		log.Error("failed to generate rename query", slog.String("error", err.Error()))
		return err
	}

	if err := a.DWHProvider.RenameColumn(ctx, renameQuery); err != nil {
		log.Error("failed to rename column in DWH", slog.String("error", err.Error()))
		return err
	}

	if err := renameColumnInView(&view, suggestion); err != nil {
		log.Error("failed to rename column in view", slog.String("error", err.Error()))
		return err
	}

	if err := a.SchemaProvider.UpdateView(ctx, view, int(suggestion.SchemaID)); err != nil {
		log.Error("failed to update view", slog.String("error", err.Error()))
		return err
	}

	if err := a.RenameSuggestionStorage.DeleteSuggestionByID(ctx, id); err != nil {
		log.Error("failed to delete suggestion", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *AnalyticsDataCenterService) RejectColumnRenameSuggestion(ctx context.Context, id int64) error {
	const op = "AnalyticsDataCenterService.RejectColumnRenameSuggestion"
	log := a.log.With(slog.String("op", op), slog.Int64("suggestion_id", id))

	if err := a.RenameSuggestionStorage.DeleteSuggestionByID(ctx, id); err != nil {
		if errors.Is(err, storage.ErrSuggestionNotFound) {
			return err
		}
		log.Error("failed to delete suggestion", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func renameColumnInView(view *models.View, suggestion models.ColumnRenameSuggestion) error {
	tableFound := false
	columnRenamed := false

	for srcIdx := range view.Sources {
		source := &view.Sources[srcIdx]
		if !strings.EqualFold(source.Name, suggestion.DatabaseName) {
			continue
		}

		for schIdx := range source.Schemas {
			schema := &source.Schemas[schIdx]
			if !strings.EqualFold(schema.Name, suggestion.SchemaName) {
				continue
			}

			for tblIdx := range schema.Tables {
				table := &schema.Tables[tblIdx]
				if !strings.EqualFold(table.Name, suggestion.TableName) {
					continue
				}

				tableFound = true
				for colIdx := range table.Columns {
					column := &table.Columns[colIdx]
					if strings.EqualFold(column.Name, suggestion.OldColumnName) {
						column.Name = suggestion.NewColumnName
						columnRenamed = true
						break
					}
				}

				if columnRenamed {
					break
				}
			}
		}
	}

	if !tableFound {
		return fmt.Errorf("table %s not found in view", suggestion.TableName)
	}

	if !columnRenamed {
		return fmt.Errorf("column %s not found in view", suggestion.OldColumnName)
	}

	return nil
}
