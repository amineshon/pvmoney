package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"pvmoney/internal/models"

	"github.com/google/uuid"
)

func assetValue(qty float64, unitValue int64) int64 {
	if qty <= 0 {
		qty = 1
	}
	return int64(math.Round(qty * float64(unitValue)))
}

func defaultAssetColor(t string) string {
	switch t {
	case "gold", "gold18", "gold24":
		return "#e0c36a"
	case "usd":
		return "#3ee0a2"
	case "eur":
		return "#7dd3fc"
	case "btc":
		return "#f59e0b"
	case "eth":
		return "#a78bfa"
	case "house":
		return "#60a5fa"
	case "car":
		return "#fb7185"
	case "land":
		return "#34d399"
	case "stock":
		return "#f9a8d4"
	default:
		return "#c9a227"
	}
}

func defaultUnit(t, unit string) string {
	if strings.TrimSpace(unit) != "" {
		return unit
	}
	switch t {
	case "gold", "gold18", "gold24":
		return "گرم"
	case "usd":
		return "دلار"
	case "eur":
		return "یورو"
	case "btc":
		return "بیت‌کوین"
	case "eth":
		return "اتریوم"
	case "house":
		return "باب"
	case "car":
		return "دستگاه"
	case "land":
		return "متر"
	case "stock":
		return "سهم"
	default:
		return "عدد"
	}
}

func (s *Store) ListAssets(ctx context.Context) ([]models.Asset, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT a.id, a.name, a.type, a.quantity, a.unit, a.unit_value, a.value, a.notes, a.color, a.source_project_id, a.created_at, a.updated_at,
       COALESCE(p.name,'')
FROM assets a
LEFT JOIN projects p ON p.id = a.source_project_id
ORDER BY a.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Asset, 0)
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAsset(ctx context.Context, id string) (models.Asset, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT a.id, a.name, a.type, a.quantity, a.unit, a.unit_value, a.value, a.notes, a.color, a.source_project_id, a.created_at, a.updated_at,
       COALESCE(p.name,'')
FROM assets a
LEFT JOIN projects p ON p.id = a.source_project_id
WHERE a.id=$1`, id)
	a, err := scanAsset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Asset{}, ErrNotFound
	}
	return a, err
}

func (s *Store) CreateAsset(ctx context.Context, in models.AssetInput) (models.Asset, error) {
	if strings.TrimSpace(in.Name) == "" {
		return models.Asset{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if in.Type == "" {
		in.Type = "other"
	}
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	if in.Color == "" {
		in.Color = defaultAssetColor(in.Type)
	}
	in.Unit = defaultUnit(in.Type, in.Unit)
	if err := CheckMoney(in.UnitValue); err != nil {
		return models.Asset{}, err
	}
	id := uuid.NewString()
	val := assetValue(in.Quantity, in.UnitValue)
	if err := CheckMoney(val); err != nil {
		return models.Asset{}, err
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO assets (id, name, type, quantity, unit, unit_value, value, notes, color, source_project_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, strings.TrimSpace(in.Name), in.Type, in.Quantity, in.Unit, in.UnitValue, val, in.Notes, in.Color, nullString(in.SourceProjectID))
	if err != nil {
		return models.Asset{}, err
	}
	if in.SourceProjectID != nil && *in.SourceProjectID != "" {
		_, _ = s.db.ExecContext(ctx, `UPDATE projects SET asset_id=$2, updated_at=NOW() WHERE id=$1`, *in.SourceProjectID, id)
	}
	return s.GetAsset(ctx, id)
}

func (s *Store) UpdateAsset(ctx context.Context, id string, in models.AssetInput) (models.Asset, error) {
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	if in.Color == "" {
		in.Color = defaultAssetColor(in.Type)
	}
	in.Unit = defaultUnit(in.Type, in.Unit)
	if err := CheckMoney(in.UnitValue); err != nil {
		return models.Asset{}, err
	}
	val := assetValue(in.Quantity, in.UnitValue)
	if err := CheckMoney(val); err != nil {
		return models.Asset{}, err
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE assets SET name=$2, type=$3, quantity=$4, unit=$5, unit_value=$6, value=$7, notes=$8, color=$9, updated_at=NOW()
WHERE id=$1`, id, strings.TrimSpace(in.Name), in.Type, in.Quantity, in.Unit, in.UnitValue, val, in.Notes, in.Color)
	if err != nil {
		return models.Asset{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.Asset{}, ErrNotFound
	}
	return s.GetAsset(ctx, id)
}

func (s *Store) DeleteAsset(ctx context.Context, id string) error {
	_, _ = s.db.ExecContext(ctx, `UPDATE projects SET asset_id=NULL WHERE asset_id=$1`, id)
	res, err := s.db.ExecContext(ctx, `DELETE FROM assets WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ConvertProjectToAsset(ctx context.Context, projectID string, in models.ConvertAssetInput) (models.Asset, error) {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return models.Asset{}, err
	}
	if in.Name == "" {
		in.Name = p.Name
	}
	if in.Type == "" {
		in.Type = p.ResultAssetType
	}
	if in.Type == "" {
		in.Type = "other"
	}
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	if in.UnitValue <= 0 {
		in.UnitValue = p.CurrentAmount
	}
	if in.Color == "" {
		in.Color = defaultAssetColor(in.Type)
	}
	pid := projectID
	if p.AssetID != nil {
		return s.UpdateAsset(ctx, *p.AssetID, models.AssetInput{
			Name:      in.Name,
			Type:      in.Type,
			Quantity:  in.Quantity,
			Unit:      in.Unit,
			UnitValue: in.UnitValue,
			Notes:     in.Notes,
			Color:     in.Color,
		})
	}
	return s.CreateAsset(ctx, models.AssetInput{
		Name:            in.Name,
		Type:            in.Type,
		Quantity:        in.Quantity,
		Unit:            in.Unit,
		UnitValue:       in.UnitValue,
		Notes:           in.Notes,
		Color:           in.Color,
		SourceProjectID: &pid,
	})
}

func scanAsset(s scanner) (models.Asset, error) {
	var a models.Asset
	var src sql.NullString
	err := s.Scan(&a.ID, &a.Name, &a.Type, &a.Quantity, &a.Unit, &a.UnitValue, &a.Value, &a.Notes, &a.Color, &src, &a.CreatedAt, &a.UpdatedAt, &a.SourceProject)
	if src.Valid {
		a.SourceProjectID = &src.String
	}
	return a, err
}
