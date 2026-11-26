package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type SocialHandler struct {
	repo  domain.SocialFeedRepository
	users domain.UserRepository
}

func NewSocialHandler(repo domain.SocialFeedRepository, users domain.UserRepository) *SocialHandler {
	if repo == nil {
		repo = NewMemorySocialRepository()
	}
	return &SocialHandler{repo: repo, users: users}
}

type createSocialPostRequest struct {
	Content   string `json:"contenido"`
	ImageURL  string `json:"imagen_url,omitempty"`
	AuctionID string `json:"subasta_id,omitempty"`
	Tag       string `json:"etiqueta,omitempty"`
}

type createSocialCommentRequest struct {
	Text string `json:"comentario"`
}

func (h *SocialHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.repo.ListPosts(r.Context(), 30)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load social feed"})
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func (h *SocialHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	authUser, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	var req createSocialPostRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" || len(content) > 2000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "post content is required"})
		return
	}

	authorName := "Participante Verificado"
	authorRole := authUser.Role
	if h.users != nil {
		if u, err := h.users.FindByID(r.Context(), authUser.ID); err == nil && u != nil {
			authorName = u.Name
			authorRole = u.Role
		}
	}

	tag := strings.TrimSpace(req.Tag)
	if tag == "" {
		tag = "COMUNIDAD"
	}

	post := domain.SocialPost{
		ID:         randomSocialID("soc_"),
		AuthorID:   authUser.ID,
		AuthorName: authorName,
		AuthorRole: authorRole,
		Content:    content,
		ImageURL:   strings.TrimSpace(req.ImageURL),
		AuctionID:  strings.TrimSpace(req.AuctionID),
		Tag:        strings.ToUpper(tag),
		LikesCount: 0,
		CreatedAt:  time.Now(),
		Comments:   []domain.SocialComment{},
	}

	created, err := h.repo.CreatePost(r.Context(), post)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not publish post"})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *SocialHandler) LikePost(w http.ResponseWriter, r *http.Request) {
	postID := strings.TrimSpace(r.PathValue("id"))
	if postID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "post id required"})
		return
	}
	count, err := h.repo.LikePost(r.Context(), postID)
	if err != nil {
		if errors.Is(err, domain.ErrSubastaNoEncontrada) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "post not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not register reaction"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"post_id": postID, "likes_count": count})
}

func (h *SocialHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	authUser, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	postID := strings.TrimSpace(r.PathValue("id"))
	var req createSocialCommentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	txt := strings.TrimSpace(req.Text)
	if postID == "" || txt == "" || len(txt) > 800 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "comment text required"})
		return
	}

	authorName := "Participante"
	if h.users != nil {
		if u, err := h.users.FindByID(r.Context(), authUser.ID); err == nil && u != nil {
			authorName = u.Name
		}
	}

	created, err := h.repo.AddComment(r.Context(), domain.SocialComment{
		ID:         randomSocialID("cmt_"),
		PostID:     postID,
		AuthorName: authorName,
		Text:       txt,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not add comment"})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func randomSocialID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + hex.EncodeToString(b[:])
}

// MemorySocialRepository ofrece almacenamiento en memoria seguro para pruebas unitarias.
type MemorySocialRepository struct {
	mu    sync.Mutex
	posts []domain.SocialPost
}

func NewMemorySocialRepository() *MemorySocialRepository {
	return &MemorySocialRepository{
		posts: []domain.SocialPost{
			{
				ID:         "soc_seed_mem",
				AuthorID:   "usr_admin",
				AuthorName: "Curaduría Aureum",
				AuthorRole: domain.RoleAuctioneer,
				Content:    "Bienvenidos al muro social de coleccionistas y vendedores.",
				Tag:        "COMUNIDAD",
				LikesCount: 3,
				CreatedAt:  time.Now(),
				Comments:   []domain.SocialComment{},
			},
		},
	}
}

func (m *MemorySocialRepository) ListPosts(_ context.Context, _ int) ([]domain.SocialPost, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.SocialPost, len(m.posts))
	copy(out, m.posts)
	return out, nil
}

func (m *MemorySocialRepository) CreatePost(_ context.Context, post domain.SocialPost) (domain.SocialPost, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.posts = append([]domain.SocialPost{post}, m.posts...)
	return post, nil
}

func (m *MemorySocialRepository) LikePost(_ context.Context, postID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.posts {
		if m.posts[i].ID == postID {
			m.posts[i].LikesCount++
			return m.posts[i].LikesCount, nil
		}
	}
	return 0, domain.ErrSubastaNoEncontrada
}

func (m *MemorySocialRepository) AddComment(_ context.Context, c domain.SocialComment) (domain.SocialComment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.posts {
		if m.posts[i].ID == c.PostID {
			m.posts[i].Comments = append(m.posts[i].Comments, c)
			return c, nil
		}
	}
	return domain.SocialComment{}, domain.ErrSubastaNoEncontrada
}
