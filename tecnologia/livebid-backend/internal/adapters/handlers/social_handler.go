package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type SocialHandler struct {
	repo         domain.SocialFeedRepository
	roomChatRepo domain.RoomChatRepository
	users        domain.UserRepository
	auctions     domain.SubastaRepository
	notifier     domain.RoomSocialNotifier
}

func NewSocialHandler(repo domain.SocialFeedRepository, users domain.UserRepository) *SocialHandler {
	if repo == nil {
		repo = NewMemorySocialRepository()
	}
	h := &SocialHandler{repo: repo, users: users}
	if rc, ok := repo.(domain.RoomChatRepository); ok {
		h.roomChatRepo = rc
	}
	return h
}

func (h *SocialHandler) WithRoomRealtime(auctions domain.SubastaRepository, notifier domain.RoomSocialNotifier) *SocialHandler {
	h.auctions = auctions
	h.notifier = notifier
	return h
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

type createRoomMessageRequest struct {
	Kind string `json:"tipo,omitempty"`
	Text string `json:"mensaje"`
}

// SellerStorefrontView expone únicamente datos públicos y de reputación de un vendedor/curador,
// omitiendo correo, teléfono, documento de identidad y dirección de billetera privada.
type SellerStorefrontView struct {
	ID              string              `json:"id"`
	Name            string              `json:"nombre"`
	Role            domain.UserRole     `json:"rol"`
	Verified        bool                `json:"verificado"`
	Bio             string              `json:"bio"`
	MemberSince     time.Time           `json:"miembro_desde"`
	TotalLikes      int                 `json:"reputacion_likes"`
	PublishedPosts  []domain.SocialPost `json:"publicaciones"`
	RelatedAuctions []*domain.Subasta   `json:"bienes_subastados"`
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
	if _, ok := CurrentUser(r); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
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

// GetSellerStorefront devuelve la vitrina pública sanitizada de un vendedor, martillero o curador.
func (h *SocialHandler) GetSellerStorefront(w http.ResponseWriter, r *http.Request) {
	sellerID := strings.TrimSpace(r.PathValue("id"))
	if sellerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "seller id required"})
		return
	}

	view := SellerStorefrontView{
		ID:              sellerID,
		Name:            "Vendedor Verificado Aureum",
		Role:            domain.RoleSeller,
		Verified:        true,
		Bio:             "Consignante acreditado de bienes y piezas de colección.",
		MemberSince:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		PublishedPosts:  []domain.SocialPost{},
		RelatedAuctions: []*domain.Subasta{},
	}

	if h.users != nil {
		if u, err := h.users.FindByID(r.Context(), sellerID); err == nil && u != nil {
			view.Name = u.Name
			view.Role = u.Role
			view.Verified = u.KYCApproved
			if strings.TrimSpace(u.Bio) != "" {
				view.Bio = u.Bio
			}
			if !u.CreatedAt.IsZero() {
				view.MemberSince = u.CreatedAt
			}
		}
	}

	linkedAuctionIDs := make(map[string]bool)
	if allPosts, err := h.repo.ListPosts(r.Context(), 50); err == nil {
		for _, p := range allPosts {
			if p.AuthorID == sellerID {
				if view.Name == "Vendedor Verificado Aureum" && p.AuthorName != "" {
					view.Name = p.AuthorName
					view.Role = p.AuthorRole
				}
				view.PublishedPosts = append(view.PublishedPosts, p)
				view.TotalLikes += p.LikesCount
				if p.AuctionID != "" {
					linkedAuctionIDs[p.AuctionID] = true
				}
			}
		}
	}

	if h.auctions != nil {
		if allAuctions, err := h.auctions.Listar(r.Context()); err == nil {
			lowerName := strings.ToLower(view.Name)
			for _, sub := range allAuctions {
				inspector := strings.ToLower(sub.DetallesLote.InspeccionadoPor)
				if linkedAuctionIDs[sub.ID] || sub.MartilleroID == sellerID || (lowerName != "" && strings.Contains(inspector, lowerName)) {
					view.RelatedAuctions = append(view.RelatedAuctions, sub)
				}
			}
			// Si es una cuenta semilla de curaduría sin coincidencia directa, mostrar el primer lote activo de referencia
			if len(view.RelatedAuctions) == 0 && len(allAuctions) > 0 && strings.HasPrefix(sellerID, "usr_seed") {
				view.RelatedAuctions = append(view.RelatedAuctions, allAuctions[0])
			}
		}
	}

	writeJSON(w, http.StatusOK, view)
}

// ListRoomMessages devuelve el historial reciente del chat en vivo de la sala.
func (h *SocialHandler) ListRoomMessages(w http.ResponseWriter, r *http.Request) {
	auctionID := strings.TrimSpace(r.PathValue("id"))
	if auctionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "auction id required"})
		return
	}
	if h.roomChatRepo == nil {
		writeJSON(w, http.StatusOK, []domain.RoomMessage{})
		return
	}
	msgs, err := h.roomChatRepo.ListRoomMessages(r.Context(), auctionID, 35)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load room messages"})
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

// PostRoomMessage registra un mensaje o reacción rápida en la sala y lo transmite por WebSocket.
func (h *SocialHandler) PostRoomMessage(w http.ResponseWriter, r *http.Request) {
	authUser, ok := CurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	auctionID := strings.TrimSpace(r.PathValue("id"))
	if auctionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "auction id required"})
		return
	}
	var req createRoomMessageRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	kind := strings.ToUpper(strings.TrimSpace(req.Kind))
	if kind == "" {
		kind = "CHAT"
	}
	if kind != "CHAT" && kind != "REACTION" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid message kind"})
		return
	}

	txt := strings.TrimSpace(req.Text)
	if txt == "" || len(txt) > 280 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message text required (max 280 chars)"})
		return
	}

	label := paddleLabelForUser(authUser.ID, authUser.Role)
	msg := domain.RoomMessage{
		ID:          randomSocialID("rm_"),
		AuctionID:   auctionID,
		AuthorLabel: label,
		Kind:        kind,
		Text:        txt,
		CreatedAt:   time.Now().UTC(),
	}

	if kind == "CHAT" && h.roomChatRepo != nil {
		saved, err := h.roomChatRepo.SaveRoomMessage(r.Context(), msg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save room message"})
			return
		}
		msg = saved
	}

	if h.notifier != nil {
		_ = h.notifier.BroadcastEventoSocialSala(r.Context(), auctionID, msg)
	}

	writeJSON(w, http.StatusCreated, msg)
}

func paddleLabelForUser(userID string, role domain.UserRole) string {
	if role == domain.RoleAuctioneer || role == domain.RoleAdmin {
		return "Martillero en Sala"
	}
	h := 0
	for i := 0; i < len(userID); i++ {
		h = (h*31 + int(userID[i])) % 900
	}
	return fmt.Sprintf("Paleta #%d", 100+h)
}

func randomSocialID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + hex.EncodeToString(b[:])
}

// MemorySocialRepository ofrece almacenamiento en memoria seguro para pruebas unitarias.
type MemorySocialRepository struct {
	mu       sync.Mutex
	posts    []domain.SocialPost
	roomMsgs map[string][]domain.RoomMessage
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
		roomMsgs: make(map[string][]domain.RoomMessage),
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

func (m *MemorySocialRepository) ListRoomMessages(_ context.Context, auctionID string, _ int) ([]domain.RoomMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	src := m.roomMsgs[auctionID]
	out := make([]domain.RoomMessage, len(src))
	copy(out, src)
	return out, nil
}

func (m *MemorySocialRepository) SaveRoomMessage(_ context.Context, msg domain.RoomMessage) (domain.RoomMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roomMsgs[msg.AuctionID] = append(m.roomMsgs[msg.AuctionID], msg)
	return msg, nil
}
