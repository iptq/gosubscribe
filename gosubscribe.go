// gosubscribe is a CRUD library for managing subscriptions.
package gosubscribe

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var DB *gorm.DB // Connect must be called before using this.

// Connect connects to a given PostreSQL database.
func Connect(host, user, dbname, password string) {
	db, err := gorm.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s user=%s dbname=%s sslmode=disable password=%s",
			host, user, dbname, password,
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&User{}, &Mapper{}, &Map{}, &Subscription{})
	DB = db // From now on, we can access the database from anywhere via DB.
}

// Subscribe subscribes a user to a list of mappers.
func (user *User) Subscribe(mappers []Mapper) {
	for _, mapper := range mappers {
		DB.Table("subscriptions").Create(&Subscription{user.ID, mapper.ID})
	}
}

// Unsubscribe unsubscribes a user from a list of mappers.
func (user *User) Unsubscribe(mappers []Mapper) {
	for _, mapper := range mappers {
		DB.Delete(&Subscription{user.ID, mapper.ID})
	}
}

// Purge unsubscribes a user from all mappers.
func (user *User) Purge() {
	DB.Table("subscriptions").Where("user_id = ?", user.ID).Delete(Subscription{})
}

// ListSubscribed gets all mappers that a user is subscribed to.
func (user *User) ListSubscribed() []Mapper {
	var mappers []Mapper
	DB.Table("subscriptions").Joins(
		"inner join mappers on subscriptions.mapper_id = mappers.id",
	).Select("mappers.id, mappers.username").Find(&mappers)
	return mappers
}

// Count gets the number of users that are subscribed to a mapper.
func (mapper *Mapper) Count() uint {
	var count uint
	DB.Model(&Subscription{}).Where("mapper_id = ?", mapper.ID).Count(&count)
	return count
}

// Count gets the subscriber counts for a list of mappers.
func Count(mappers []Mapper) map[Mapper]uint {
	counts := make(map[Mapper]uint)
	for _, mapper := range mappers {
		counts[mapper] = mapper.Count()
	}
	return counts
}

// GetSecret retrieve's a user's secret.
func GetSecret(user User) (string, error) {
	if len(user.Secret) != 0 {
		return user.Secret, nil
	} else {
		return "", errors.New("You don't have a secret; run `.init` to get one.")
	}
}

// UserFromSecret retrieves a user from their unique secret.
func UserFromSecret(secret string) (User, error) {
	var user User
	DB.Where("secret = ?", secret).First(&user)
	if user.ID == 0 {
		return user, errors.New("Incorrect secret.")
	}
	return user, nil
}

// GenSecret generates a "random" string of digits.
func GenSecret() string {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprint(time.Now().UnixNano())))
	return fmt.Sprint(h.Sum32())

}
