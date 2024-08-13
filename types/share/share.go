package share

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/celestiaorg/nmt"

	"github.com/celestiaorg/celestia-openrpc/types/appconsts"
	"github.com/celestiaorg/celestia-openrpc/types/core"
	"github.com/celestiaorg/celestia-openrpc/types/namespace"
)

// Root represents root commitment to multiple Shares.
// In practice, it is a commitment to all the Data in a square.
type Root = core.DataAvailabilityHeader

// NamespacedRow represents all shares with proofs within a specific namespace of a single EDS row.
type NamespacedRow struct {
	Shares []Share    `json:"shares"`
	Proof  *nmt.Proof `json:"proof"`
}

// NamespacedShares represents all shares with proofs within a specific namespace of an EDS.
type NamespacedShares []NamespacedRow

var (
	// DefaultRSMT2DCodec sets the default rsmt2d.Codec for shares.
	DefaultRSMT2DCodec = appconsts.DefaultCodec
)

const (
	// Size is a system-wide size of a share, including both data and namespace GetNamespace
	Size = appconsts.ShareSize
)

// Share contains the raw share data without the corresponding namespace.
// NOTE: Alias for the byte is chosen to keep maximal compatibility, especially with rsmt2d.
// Ideally, we should define reusable type elsewhere and make everyone(Core, rsmt2d, ipld) to rely
// on it.
// Share contains the raw share data (including namespace ID).
type Share struct {
	data []byte
}

func (s *Share) UnmarshalJSON(data []byte) error {
	share, err := NewShare(data)
	if err != nil {
		return err
	}
	s.data = share.data
	return nil
}

func (s *Share) Namespace() (namespace.Namespace, error) {
	if len(s.data) < appconsts.NamespaceSize {
		panic(fmt.Sprintf("share %s is too short to contain a namespace", s))
	}
	return namespace.From(s.data[:appconsts.NamespaceSize])
}

func (s *Share) InfoByte() (InfoByte, error) {
	if len(s.data) < namespace.NamespaceSize+appconsts.ShareInfoBytes {
		return 0, fmt.Errorf("share %s is too short to contain an info byte", s)
	}
	// the info byte is the first byte after the namespace
	unparsed := s.data[namespace.NamespaceSize]
	return ParseInfoByte(unparsed)
}

func NewShare(data []byte) (*Share, error) {
	if err := validateSize(data); err != nil {
		return nil, err
	}
	return &Share{data}, nil
}

func (s *Share) Validate() error {
	return validateSize(s.data)
}

func validateSize(data []byte) error {
	if len(data) != appconsts.ShareSize {
		return fmt.Errorf("share data must be %d bytes, got %d", appconsts.ShareSize, len(data))
	}
	return nil
}

func (s *Share) Len() int {
	return len(s.data)
}

func (s *Share) Version() (uint8, error) {
	infoByte, err := s.InfoByte()
	if err != nil {
		return 0, err
	}
	return infoByte.Version(), nil
}

func (s *Share) DoesSupportVersions(supportedShareVersions []uint8) error {
	ver, err := s.Version()
	if err != nil {
		return err
	}
	if !bytes.Contains(supportedShareVersions, []byte{ver}) {
		return fmt.Errorf("unsupported share version %v is not present in the list of supported share versions %v", ver, supportedShareVersions)
	}
	return nil
}

// IsSequenceStart returns true if this is the first share in a sequence.
func (s *Share) IsSequenceStart() (bool, error) {
	infoByte, err := s.InfoByte()
	if err != nil {
		return false, err
	}
	return infoByte.IsSequenceStart(), nil
}

// IsCompactShare returns true if this is a compact share.
func (s Share) IsCompactShare() (bool, error) {
	ns, err := s.Namespace()
	if err != nil {
		return false, err
	}
	isCompact := ns.IsTx() || ns.IsPayForBlob()
	return isCompact, nil
}

// SequenceLen returns the sequence length of this *share and optionally an
// error. It returns 0, nil if this is a continuation share (i.e. doesn't
// contain a sequence length).
func (s *Share) SequenceLen() (sequenceLen uint32, err error) {
	isSequenceStart, err := s.IsSequenceStart()
	if err != nil {
		return 0, err
	}
	if !isSequenceStart {
		return 0, nil
	}

	start := appconsts.NamespaceSize + appconsts.ShareInfoBytes
	end := start + appconsts.SequenceLenBytes
	if len(s.data) < end {
		return 0, fmt.Errorf("share %s with length %d is too short to contain a sequence length",
			s, len(s.data))
	}
	return binary.BigEndian.Uint32(s.data[start:end]), nil
}

// IsPadding returns whether this *share is padding or not.
func (s *Share) IsPadding() (bool, error) {
	isNamespacePadding, err := s.isNamespacePadding()
	if err != nil {
		return false, err
	}
	isTailPadding, err := s.isTailPadding()
	if err != nil {
		return false, err
	}
	isReservedPadding, err := s.isReservedPadding()
	if err != nil {
		return false, err
	}
	return isNamespacePadding || isTailPadding || isReservedPadding, nil
}

func (s *Share) isNamespacePadding() (bool, error) {
	isSequenceStart, err := s.IsSequenceStart()
	if err != nil {
		return false, err
	}
	sequenceLen, err := s.SequenceLen()
	if err != nil {
		return false, err
	}

	return isSequenceStart && sequenceLen == 0, nil
}

func (s *Share) isTailPadding() (bool, error) {
	ns, err := s.Namespace()
	if err != nil {
		return false, err
	}
	return ns.IsTailPadding(), nil
}

func (s *Share) isReservedPadding() (bool, error) {
	ns, err := s.Namespace()
	if err != nil {
		return false, err
	}
	return ns.IsReservedPadding(), nil
}

func (s *Share) ToBytes() []byte {
	return s.data
}

// RawData returns the raw share data. The raw share data does not contain the
// namespace ID, info byte, sequence length, or reserved bytes.
func (s *Share) RawData() (rawData []byte, err error) {
	if len(s.data) < s.rawDataStartIndex() {
		return rawData, fmt.Errorf("share %s is too short to contain raw data", s)
	}

	return s.data[s.rawDataStartIndex():], nil
}

func (s *Share) rawDataStartIndex() int {
	isStart, err := s.IsSequenceStart()
	if err != nil {
		panic(err)
	}
	isCompact, err := s.IsCompactShare()
	if err != nil {
		panic(err)
	}

	index := appconsts.NamespaceSize + appconsts.ShareInfoBytes
	if isStart {
		index += appconsts.SequenceLenBytes
	}
	if isCompact {
		index += appconsts.CompactShareReservedBytes
	}
	return index
}

// RawDataWithReserved returns the raw share data while taking reserved bytes into account.
func (s *Share) RawDataUsingReserved() (rawData []byte, err error) {
	rawDataStartIndexUsingReserved, err := s.rawDataStartIndexUsingReserved()
	if err != nil {
		return nil, err
	}

	// This means share is the last share and does not have any transaction beginning in it
	if rawDataStartIndexUsingReserved == 0 {
		return []byte{}, nil
	}
	if len(s.data) < rawDataStartIndexUsingReserved {
		return rawData, fmt.Errorf("share %s is too short to contain raw data", s)
	}

	return s.data[rawDataStartIndexUsingReserved:], nil
}

// rawDataStartIndexUsingReserved returns the start index of raw data while accounting for
// reserved bytes, if it exists in the share.
func (s *Share) rawDataStartIndexUsingReserved() (int, error) {
	isStart, err := s.IsSequenceStart()
	if err != nil {
		return 0, err
	}
	isCompact, err := s.IsCompactShare()
	if err != nil {
		return 0, err
	}

	index := appconsts.NamespaceSize + appconsts.ShareInfoBytes
	if isStart {
		index += appconsts.SequenceLenBytes
	}

	if isCompact {
		reservedBytes, err := ParseReservedBytes(s.data[index : index+appconsts.CompactShareReservedBytes])
		if err != nil {
			return 0, err
		}
		return int(reservedBytes), nil
	}
	return index, nil
}

func ToBytes(shares []Share) (bytes [][]byte) {
	bytes = make([][]byte, len(shares))
	for i, share := range shares {
		bytes[i] = []byte(share.data)
	}
	return bytes
}

func FromBytes(bytes [][]byte) (shares []Share, err error) {
	for _, b := range bytes {
		share, err := NewShare(b)
		if err != nil {
			return nil, err
		}
		shares = append(shares, *share)
	}
	return shares, nil
}

// DataHash is a representation of the Root hash.
type DataHash []byte

func (dh DataHash) Validate() error {
	if len(dh) != 32 {
		return fmt.Errorf("invalid hash size, expected 32, got %d", len(dh))
	}
	return nil
}

func (dh DataHash) String() string {
	return fmt.Sprintf("%X", []byte(dh))
}
