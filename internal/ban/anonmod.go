package ban

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// Issue type keys for anonmod identities.
const (
	AnonmodIssueTypeReport = "report"
	AnonmodIssueTypeAppeal = "appeal"
)

// AnonmodEnabledProvider reports whether the anonymized moderator feature is
// currently enabled. Implemented by *config.Configuration.
type AnonmodEnabledProvider interface {
	AnonmodEnabled() bool
}

// AnonIdentity is the stable anonymous persona assigned to a moderator
// author for a single issue (a report or a ban appeal). A moderator keeps
// the same persona across every reply they make within the issue, and each
// issue gets its own independently assigned set.
type AnonIdentity struct {
	IssueType      string
	IssueID        int32
	AuthorID       steamid.SteamID
	AnonName       string
	AnonAvatarHash string
}

// AnonmodRepository persists per-issue anonymous personas.
type AnonmodRepository struct {
	database.Database
}

func NewAnonmodRepository(database database.Database) AnonmodRepository {
	return AnonmodRepository{Database: database}
}

func (r AnonmodRepository) GetIdentity(ctx context.Context, issueType string, issueID int32, authorID steamid.SteamID) (AnonIdentity, error) {
	row := r.QueryRow(ctx,
		`SELECT issue_type, issue_id, author_id, anon_name, anon_avatar_hash
		 FROM anonmod_identity
		 WHERE issue_type = $1 AND issue_id = $2 AND author_id = $3`,
		issueType, issueID, authorID.Int64(),
	)

	var (
		identity AnonIdentity
		author   int64
	)

	if errScan := row.Scan(&identity.IssueType, &identity.IssueID, &author, &identity.AnonName, &identity.AnonAvatarHash); errScan != nil {
		return identity, database.Err(errScan)
	}

	identity.AuthorID = steamid.New(author)

	return identity, nil
}

func (r AnonmodRepository) Identities(ctx context.Context, issueType string, issueID int32) ([]AnonIdentity, error) {
	rows, errQuery := r.Query(ctx,
		`SELECT issue_type, issue_id, author_id, anon_name, anon_avatar_hash
		 FROM anonmod_identity
		 WHERE issue_type = $1 AND issue_id = $2`,
		issueType, issueID,
	)
	if errQuery != nil {
		return nil, database.Err(errQuery)
	}

	defer rows.Close()

	var identities []AnonIdentity

	for rows.Next() {
		var (
			identity AnonIdentity
			authorID int64
		)

		if errScan := rows.Scan(&identity.IssueType, &identity.IssueID, &authorID, &identity.AnonName, &identity.AnonAvatarHash); errScan != nil {
			return nil, errors.Join(errScan, database.ErrScanResult)
		}

		identity.AuthorID = steamid.New(authorID)

		identities = append(identities, identity)
	}

	return identities, nil
}

func (r AnonmodRepository) SaveIdentity(ctx context.Context, identity AnonIdentity) error {
	query := r.Builder().
		Insert("anonmod_identity").
		Columns("issue_type", "issue_id", "author_id", "anon_name", "anon_avatar_hash").
		Values(identity.IssueType, identity.IssueID, identity.AuthorID.Int64(), identity.AnonName, identity.AnonAvatarHash).
		Suffix("ON CONFLICT (issue_type, issue_id, author_id) DO NOTHING")

	return database.Err(r.ExecInsertBuilder(ctx, query))
}

// Anonmod assigns and retrieves per-issue anonymous personas for moderator
// replies to reports and ban appeals.
type Anonmod struct {
	repo    AnonmodRepository
	enabled AnonmodEnabledProvider
}

func NewAnonmod(repo AnonmodRepository, enabled AnonmodEnabledProvider) Anonmod {
	return Anonmod{repo: repo, enabled: enabled}
}

// Enabled reports whether the anonymized moderator feature is on.
func (a Anonmod) Enabled() bool {
	return a.enabled.AnonmodEnabled()
}

// Identity returns the persona assigned to the author for the issue, creating
// a new random one if none exists yet. Names and avatars are unique within
// the issue so different moderators are distinguishable.
func (a Anonmod) Identity(ctx context.Context, issueType string, issueID int32, authorID steamid.SteamID) (AnonIdentity, error) {
	existing, errExisting := a.repo.GetIdentity(ctx, issueType, issueID, authorID)
	if errExisting == nil {
		return existing, nil
	}

	if !errors.Is(errExisting, database.ErrNoResult) {
		return AnonIdentity{}, errExisting
	}

	existingAll, errAll := a.repo.Identities(ctx, issueType, issueID)
	if errAll != nil {
		return AnonIdentity{}, errAll
	}

	usedNames := make(map[string]struct{}, len(existingAll))
	usedAvatars := make(map[string]struct{}, len(existingAll))
	for _, entry := range existingAll {
		usedNames[strings.ToLower(entry.AnonName)] = struct{}{}
		usedAvatars[entry.AnonAvatarHash] = struct{}{}
	}

	identity := AnonIdentity{
		IssueType:      issueType,
		IssueID:        issueID,
		AuthorID:       authorID,
		AnonName:       randomAnonName(usedNames),
		AnonAvatarHash: randomAnonAvatar(usedAvatars),
	}

	if errSave := a.repo.SaveIdentity(ctx, identity); errSave != nil {
		return AnonIdentity{}, errSave
	}

	// A concurrent create may have won the race; return the stored row.
	return a.repo.GetIdentity(ctx, issueType, issueID, authorID)
}

var (
	//nolint:gochecknoglobals // Static persona pools for random assignment.
	anonFirstNamePool = []string{
		"Alex", "Avery", "Bailey", "Casey", "Charlie", "Dakota",
		"Ellis", "Finley", "Harper", "Indy", "Jules", "Knox",
		"Lane", "Marlow", "Morgan", "Nick",
		"Oakley", "Parker", "Quinn", "Riley", "Sage", "Teddy",
		"Wells", "Ziggy",
	}

	//nolint:gochecknoglobals // Static persona pools for random assignment.
	anonLastNamePool = []string{
		"Barnes", "Caldwell", "Dawson", "Ellery", "Foster", "Grant",
		"Hargrove", "Iverson", "Jacobs", "Kowalski", "Larkin", "Mercer",
		"Noble", "Okafor", "Prescott", "Quill",
		"Radcliffe", "Sutton", "Tremaine", "Vance", "Whitmore", "Yates",
		"Zeller", "Ashcroft",
	}
)

// anonAvatarPool is a set of valid steam avatar hashes used for the fake
// persona pictures.
var anonAvatarPool = []string{ //nolint:gochecknoglobals // Static persona pools for random assignment.
	"4e6b4142eca38e4facdd9d3511c8b4936832c508", "00b12943a5317769d17216f71e9f32c24fea8c2e",
	"afb5ce8d8bfb3cc88289650fedb3743309ead995", "7b680a9db7f348c3cae6127e9b5189c256a5af1f",
	"d08d0b974183edc5e9ddaa09ff9c3484b51edc4b", "d00b24b23d4b2d3a5fbc72d1bbd610e890bd369b",
	"8fbc89399988aeeef2498efff385f4d41169823f", "4ce5030ce4bc3b18e87efbf0db9e5f957b6b661a",
	"ed7adf7c16c7ac648da90b60a81f807532885fce", "3ad97a32e439e2ca07e814ec04b123c08bb261f1",
	"e12c9a419e5d323e42b0491e26fa8de45238dd0b", "14789ea9d9de015983e16460b430a818b43cd1d5",
	"69d848ba8f3ef13c133042b840cfe7b79cbc69ed", "c09e2796b010c1a312eb1c759c92d2558f73e51e",
	"73e4a9f4a42d8a2f68bb601f3eea6e9b20a7c289", "71a783686fc5d1a9c68da6daba472d368bc99dfb",
	"3a0782d0c20e6d45a0b294d5183e88c4c99abc42", "b5a238ba0e45b5222728f72ede92d76a206f512b",
	"286733c780a6a999450332c57e59de2a27439951", "f032c4c320ef780126cb3e81dd924f7162a3014e",
	"67e57e45c631d96629f6370d7c81e7d7c1e1db73", "21770dc984b3ec63ea22193eb262f3c9a4996f23",
	"55a94e13d6d2071d2019eee73db08474a7beb912", "7564f5fa0197df6269ea5a3e8ea5a97c6ed530c6",
	"bf7ff9a378b3163fc7ad94f568b19525b8072748", "773fd650a55597ecbcc7212066a4986169e4e30d",
	"0315fa895909761597af5b372e4c9758a541370c", "9de4d8f9c1009e01f15557dd0d907a1752beda41",
	"db4782207e2f0cd03c76962a62ff5b400812538c", "4edbb4b8cc3443d6ce9556ecf3ebfd2880058b6f",
	"2c5346f8ef5a8897fd53b3ac796e0f2c5f0ac7ad", "a272081f942b03a046d2a08d99896020a890efb7",
	"efe409c81387980c9595561f5ee92a7f5d75b94e", "12d4080a3c7e726e58688a053375f5b60f9d4d37",
	"793cb496e7bc0b416eec4391693ec8d7fa310064", "f4dead6cd387380accb170a3c08a3b6f20880680",
	"453839bc607ad5cd6878d9b820d722104e87b6cb", "6f1ab1fba50817f28635f6f43209f5b08e4794da",
	"086a7591893019812e648f808104ac6db9604c17", "3f9d6fb608856f48a259f666185854c278d457ec",
	"bd1587f811f03dce921a96996d8f3a3741fa4231", "e976f0c6c96100dc33e0e85d813799d59ff088ae",
	"b6968f4a8e4fafbc0bc09f94bc2dd2bb8a9844ff", "05cbc87b97dca184f6433906225c26ce1eafd466",
	"a1a9a63f7b4f33ceff28a0c7520f0c69f81245ec", "038ced79e84e81950f98ce7c0506575d5edf43d0",
	"8ebc8fa3db8d5cfbe986b2ccd15532f9b89c969b", "36659e246fbd26321e08d40874f2a94692384784",
	"38724d4bcfe13c4fbac42d583d80cb33948b6dd8", "8e90b4519c919ab84343ea052ae357fb5fdbdfb8",
	"49f79589e9bd24b87e3216e6469d423d032541e9", "47f69d5d7a04094b0e2ca8b1d78a5f9145723aa9",
	"415e7d9762c958bd44d26e788b1b1bae8b891cb2", "b6207e8c8338d3ffcb8756c14d1395666d67dbea",
	"3f95a5c0d17b77bf203cf87ebd9394a97889860b", "645d926285c0268b5828d36ef351822bb7e645ca",
	"893df2368f9654df9b68f67a42f20182b9b8dfaa", "1432ad3ebc55c3b30cf3b003a6338102c918f657",
	"7ef9a3ec513fba3676136dce912ba94699c4135b", "46f448cf8809331552d9f6b70c28d51d09a37618",
	"e53f79230f526219f7acbb59b538742143aaca2a", "1c0b2e5cdf8b1425a499c9471b8e39e44942684d",
	"cc4e156f5fe1883a618416303f9621042aa6125d", "41ea63a0964c642748d687a17d82e06eb5c7eab9",
	"7ffa729b47be9fb5cd079cd42299d077f6049e16", "704524bc2e517850fa085a887b7d877b95cc4b2a",
	"95249f01d16785432474249e6b08c4b755b46cf6", "ff484601351b73ca14ea11b2a208d72fbe023a28",
	"b38ab10f10ccf56142cf865c5e48b2c5764ca592", "723f50e213fc93fe5ad1843539ff69dd89be653a",
	"f55dbb084617f4f20bc2d6a34d6f7c52ab824ea3", "e2fa239e2446ca0cea4c0caef883e84d5027d540",
	"39f33674fc92cdf445a499880114043728b69fc5", "c474bd7615580048e67a7986868396235af3f895",
	"4da94edc6750cd357ef637840b01bf40b7fcdba3", "7617a07f6a25319710690607da597d4929431d19",
	"62033f2046b8d3655b5408ece55e97a3a3d91a6d", "16209282c1c7ffdf5403d21a644ee52dde1c46c6",
	"f10c9b8547224d9cfab1d39277552c242518bab2", "b1b389cb4cef934bf959f02f83992af5f545fe6e",
}

// randomAnonName picks a plausible random personaname that is not already
// used within the issue.
func randomAnonName(used map[string]struct{}) string {
	firstCount, lastCount := len(anonFirstNamePool), len(anonLastNamePool)

	for range firstCount * lastCount {
		name := fmt.Sprintf("%s %s",
			anonFirstNamePool[rand.IntN(firstCount)], //nolint:gosec
			anonLastNamePool[rand.IntN(lastCount)],   //nolint:gosec
		)
		if _, ok := used[strings.ToLower(name)]; !ok {
			return name
		}
	}

	// Combination space exhausted; append a counter to guarantee uniqueness.
	suffix := 1
	for {
		name := fmt.Sprintf("%s %d", anonFirstNamePool[rand.IntN(firstCount)], suffix) //nolint:gosec
		if _, ok := used[strings.ToLower(name)]; !ok {
			return name
		}

		suffix++
	}
}

// randomAnonAvatar picks an avatar hash that is not already used within the issue.
func randomAnonAvatar(used map[string]struct{}) string {
	for range anonAvatarPool {
		hash := anonAvatarPool[rand.IntN(len(anonAvatarPool))] //nolint:gosec
		if _, ok := used[hash]; !ok {
			return hash
		}
	}

	// Pool exhausted; reusing a hash is preferable to blocking the reply.
	return anonAvatarPool[0]
}
