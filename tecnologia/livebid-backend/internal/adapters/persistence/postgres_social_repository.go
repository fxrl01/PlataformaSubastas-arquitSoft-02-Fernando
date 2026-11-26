package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSocialRepository struct {
	writePool *pgxpool.Pool
	readPool  *pgxpool.Pool
}

func NewPostgresSocialRepository(writePool, readPool *pgxpool.Pool) *PostgresSocialRepository {
	if readPool == nil {
		readPool = writePool
	}
	return &PostgresSocialRepository{writePool: writePool, readPool: readPool}
}

func (r *PostgresSocialRepository) ListPosts(ctx context.Context, limit int) ([]domain.SocialPost, error) {
	if limit <= 0 || limit > 60 {
		limit = 30
	}
	rows, err := r.readPool.Query(ctx, `
		SELECT id, autor_id, autor_nombre, autor_rol, contenido, imagen_url, subasta_id, etiqueta, likes_count, created_at
		FROM social_posts
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list social posts: %w", err)
	}
	defer rows.Close()

	var posts []domain.SocialPost
	postIdx := make(map[string]int)
	var ids []string

	for rows.Next() {
		var p domain.SocialPost
		var role string
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.AuthorName, &role, &p.Content, &p.ImageURL, &p.AuctionID, &p.Tag, &p.LikesCount, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan social post: %w", err)
		}
		p.AuthorRole = domain.UserRole(role)
		p.Comments = []domain.SocialComment{}
		postIdx[p.ID] = len(posts)
		ids = append(ids, p.ID)
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []domain.SocialPost{}, nil
	}

	cRows, err := r.readPool.Query(ctx, `
		SELECT id, post_id, autor_nombre, comentario, created_at
		FROM social_comments
		WHERE post_id = ANY($1)
		ORDER BY created_at ASC`, ids)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var c domain.SocialComment
			if err := cRows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.Text, &c.CreatedAt); err == nil {
				if idx, ok := postIdx[c.PostID]; ok {
					posts[idx].Comments = append(posts[idx].Comments, c)
				}
			}
		}
	}
	return posts, nil
}

func (r *PostgresSocialRepository) CreatePost(ctx context.Context, post domain.SocialPost) (domain.SocialPost, error) {
	_, err := r.writePool.Exec(ctx, `
		INSERT INTO social_posts (id, autor_id, autor_nombre, autor_rol, contenido, imagen_url, subasta_id, etiqueta, likes_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		post.ID, post.AuthorID, post.AuthorName, string(post.AuthorRole),
		post.Content, post.ImageURL, post.AuctionID, post.Tag, post.LikesCount, post.CreatedAt,
	)
	if err != nil {
		return domain.SocialPost{}, fmt.Errorf("create social post: %w", err)
	}
	if post.Comments == nil {
		post.Comments = []domain.SocialComment{}
	}
	return post, nil
}

func (r *PostgresSocialRepository) LikePost(ctx context.Context, postID string) (int, error) {
	var count int
	err := r.writePool.QueryRow(ctx, `
		UPDATE social_posts
		SET likes_count = likes_count + 1
		WHERE id = $1
		RETURNING likes_count`, postID).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrSubastaNoEncontrada
	}
	if err != nil {
		return 0, fmt.Errorf("like social post: %w", err)
	}
	return count, nil
}

func (r *PostgresSocialRepository) AddComment(ctx context.Context, comment domain.SocialComment) (domain.SocialComment, error) {
	_, err := r.writePool.Exec(ctx, `
		INSERT INTO social_comments (id, post_id, autor_nombre, comentario, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		comment.ID, comment.PostID, comment.AuthorName, comment.Text, comment.CreatedAt,
	)
	if err != nil {
		return domain.SocialComment{}, fmt.Errorf("add social comment: %w", err)
	}
	return comment, nil
}
