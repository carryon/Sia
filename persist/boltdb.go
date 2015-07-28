package persist

import (
	"errors"

	"github.com/boltdb/bolt"
)

// See json.go for definitions of errors and metadata

type BoltDatabase struct {
	*bolt.DB
	meta Metadata
}

var (
	ErrNilEntry  = errors.New("entry does not exist")
	ErrNilBucket = errors.New("bucket does not exist")
)

// checkDbMetadata confirms that the metadata in the database is
// correct. If there is no metadata, correct metadata is inserted
func (db *BoltDatabase) checkMetadata(meta Metadata) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Metadata"))
		if bucket == nil {
			err := db.updateMetadata(tx)
			if err != nil {
				return err
			}
		}

		// Get bucket in the case that it has just been made
		// Doing this twice should have no ill effect
		bucket = tx.Bucket([]byte("Metadata"))

		header := bucket.Get([]byte("Header"))
		if string(header) != meta.Header {
			return ErrBadHeader
		}

		version := bucket.Get([]byte("Version"))
		if string(version) != meta.Version {
			return ErrBadVersion
		}
		return nil
	})
	return err
}

// updateDbMetadata will set the contents of the metadata bucket to be
// what is stored inside the metadata argument
func (db *BoltDatabase) updateMetadata(tx *bolt.Tx) error {
	bucket, err := tx.CreateBucketIfNotExists([]byte("Metadata"))
	if err != nil {
		return err
	}

	err = bucket.Put([]byte("Header"), []byte(db.meta.Header))
	if err != nil {
		return err
	}

	err = bucket.Put([]byte("Version"), []byte(db.meta.Version))
	if err != nil {
		return err
	}
	return nil
}

// GetFromBucket is a wrapper around a bolt database lookup. If the
// element does not exist, no error will be thrown, but the requested
// element will be nil.
func (db *BoltDatabase) GetFromBucket(bucketName string, key []byte) ([]byte, error) {
	var bytes []byte
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return errors.New("requested bucket does not exist: " + bucketName)
		}

		// Note that bytes could be nil. This is OK, but needs
		// to be checked in calling functions
		value := bucket.Get(key)
		if value == nil {
			return ErrNilEntry
		}
		bytes = make([]byte, len(value))
		copy(bytes, value)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// Exists checks for the existance of an item in the specified bucket
func (db *BoltDatabase) Exists(bucketName string, key []byte) (bool, error) {
	var exists bool
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return errors.New("requested bucket does not exist: " + bucketName)
		}

		v := bucket.Get(key)
		exists = v != nil
		return nil
	})
	return exists, err
}

func (db *BoltDatabase) BucketSize(bucketName string) (uint64, error) {
	var size uint64
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		size = uint64(b.Stats().KeyN)
		return nil
	})
	return size, err
}

// closeDatabase saves the bolt database to a file, and updates metadata
func (db *BoltDatabase) CloseDatabase() error {
	err := db.Update(db.updateMetadata)
	if err != nil {
		return err
	}

	db.Close()
	return nil
}

// openDatabase opens a database filename and checks metadata
func OpenDatabase(meta Metadata, filename string) (*BoltDatabase, error) {
	db, err := bolt.Open(filename, 0600, nil)
	if err != nil {
		return nil, err
	}

	boltDB := &BoltDatabase{db, meta}

	err = boltDB.checkMetadata(meta)
	if err != nil {
		return nil, err
	}
	return boltDB, nil
}