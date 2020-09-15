package faunastore

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fauna/faunadb-go/v3/faunadb"
	f "github.com/fauna/faunadb-go/v3/faunadb"
	"github.com/gorilla/sessions"
)

type FaunaStore struct {
	client     *f.FaunaClient
	options    sessions.Options
	serializer GobSerializer
	keyPrefix  string
}

type FaunaSession struct {
	Id  string `fauna:"id"`
	Val []byte `fauna:"val"`
}

func NewFaunaStore(client *f.FaunaClient) (*FaunaStore, error) {
	fs := &FaunaStore{
		client: client,
		options: sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		keyPrefix:  "session:",
		serializer: GobSerializer{},
	}
	// TODO: add error here
	return fs, nil
}

// Get should return a cached session.
func (s *FaunaStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New should create and return a new session.
//
// Note that New should never return a nil session, even in the case of
// an error if using the Registry infrastructure to cache the session.
func (s *FaunaStore) New(r *http.Request, name string) (*sessions.Session, error) {

	session := sessions.NewSession(s, name)
	panic(fmt.Sprintf("%+v\n", s))
	opts := s.options
	session.Options = &opts
	session.IsNew = true

	c, err := r.Cookie(name)
	if err != nil {
		return session, nil
	}
	session.ID = c.Value

	err = s.load(session)
	if err == nil {
		session.IsNew = false
	}
	return session, err
}

// Save should persist session to the underlying store implementation.
func (s *FaunaStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Delete if max-age is <= 0
	if session.Options.MaxAge <= 0 {
		if err := s.delete(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		id, err := generateRandomKey()
		if err != nil {
			return errors.New("redisstore: failed to generate session id")
		}
		session.ID = id
	}
	if err := s.save(session); err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), session.ID, session.Options))
	return nil
}

func (s *FaunaStore) Options(opts sessions.Options) {
	s.options = opts
}

func (s *FaunaStore) load(session *sessions.Session) error {
	/*
		cmd := s.client.Get(s.keyPrefix + session.ID)
		if cmd.Err() != nil {
			return cmd.Err()
		}

		b, err := cmd.Bytes()
		if err != nil {
			return err
		}

		return s.serializer.Deserialize(b, session)
	*/
	res, err := s.client.Query(f.Get(f.MatchTerm(f.Index("sessions_by_id"), s.keyPrefix+session.ID)))
	if err != nil {
		return err
	}

	var fs FaunaSession

	if err := res.At(f.ObjKey("data")).Get(&fs); err != nil {
		return err
	}

	return s.serializer.Deserialize(fs.Val, session)

}

func (s *FaunaStore) delete(session *sessions.Session) error {
	res, err := s.client.Query(f.Get(f.MatchTerm(f.Index("sessions_by_id"), s.keyPrefix+session.ID)))
	if err != nil {
		return err
	}

	var toBeDeletedRef faunadb.RefV

	if err := res.At(f.ObjKey("ref")).Get(&toBeDeletedRef); err != nil {
		return err
	}

	_, err = s.client.Query(f.Delete(f.RefCollection(f.Collection("sessions"), toBeDeletedRef.ID)))
	return err
}

func (s *FaunaStore) save(session *sessions.Session) error {

	b, err := s.serializer.Serialize(session)
	if err != nil {
		return err
	}

	ttl := time.Now().Add(time.Duration(session.Options.MaxAge) * time.Second)

	res, err := s.client.Query(f.Get(f.MatchTerm(f.Index("sessions_by_id"), s.keyPrefix+session.ID)))
	if err == nil { // a session already exists in fauna
		var existingRef faunadb.RefV
		if err := res.At(f.ObjKey("ref")).Get(&existingRef); err != nil {
			return err
		}
		_, err = s.client.Query(
			f.Replace(
				f.RefCollection(f.Collection("sessions"), existingRef.ID),
				f.Obj{
					"data": f.Obj{
						"id":  s.keyPrefix + session.ID,
						"val": b,
					},
					"ttl": f.Time(ttl.Format(time.RFC3339)),
				},
			),
		)
		return err
	}

	_, err = s.client.Query(
		f.Create(
			f.Collection("sessions"),
			f.Obj{
				"data": f.Obj{
					"id":  s.keyPrefix + session.ID,
					"val": b,
				},
				"ttl": f.Time(ttl.Format(time.RFC3339)),
			},
		),
	)
	return err
}

type GobSerializer struct{}

func (gs GobSerializer) Serialize(s *sessions.Session) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(s.Values)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

func (gs GobSerializer) Deserialize(d []byte, s *sessions.Session) error {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	return dec.Decode(&s.Values)
}

// from https://github.com/rbcervilla/redisstore/blob/master/redisstore.go#L187
func generateRandomKey() (string, error) {
	k := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(k), "="), nil
}
