// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package arch

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	packages_model "code.gitea.io/gitea/models/packages"
	user_model "code.gitea.io/gitea/models/user"
	packages_module "code.gitea.io/gitea/modules/packages"
	arch_module "code.gitea.io/gitea/modules/packages/arch"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	packages_service "code.gitea.io/gitea/services/packages"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func GetOrCreateRepositoryVersion(ctx context.Context, ownerID int64) (*packages_model.PackageVersion, error) {
	return packages_service.GetOrCreateInternalPackageVersion(ctx, ownerID, packages_model.TypeArch, arch_module.RepositoryPackage, arch_module.RepositoryVersion)
}

func BuildAllRepositoryFiles(ctx context.Context, ownerID int64) error {
	pv, err := GetOrCreateRepositoryVersion(ctx, ownerID)
	if err != nil {
		return err
	}
	// remove old db files
	pfs, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		return err
	}
	for _, pf := range pfs {
		if strings.HasSuffix(pf.Name, ".db") {
			if err := BuildPacmanDB(ctx, ownerID, pf.CompositeKey); err != nil {
				return err
			}
		}
	}
	return nil
}

func BuildCustomRepositoryFiles(ctx context.Context, ownerID int64, disco string) error {
	pv, err := GetOrCreateRepositoryVersion(ctx, ownerID)
	if err != nil {
		return err
	}
	// remove old db files
	pfs, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		return err
	}
	for _, pf := range pfs {
		if strings.HasSuffix(pf.Name, ".db") && pf.CompositeKey == disco {
			if err := BuildPacmanDB(ctx, ownerID, pf.CompositeKey); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewFileSign(ctx context.Context, ownerID int64, input io.Reader) (*packages_module.HashedBuffer, error) {
	// If no signature is specified, it will be generated by Gitea.
	priv, _, err := GetOrCreateKeyPair(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	block, err := armor.Decode(strings.NewReader(priv))
	if err != nil {
		return nil, err
	}
	e, err := openpgp.ReadEntity(packet.NewReader(block.Body))
	if err != nil {
		return nil, err
	}
	pkgSig, err := packages_module.NewHashedBuffer()
	if err != nil {
		return nil, err
	}
	defer pkgSig.Close()
	if err := openpgp.DetachSign(pkgSig, e, input, nil); err != nil {
		return nil, err
	}
	return pkgSig, nil
}

// BuildPacmanDB Create db signature cache
func BuildPacmanDB(ctx context.Context, ownerID int64, distro string) error {
	pv, err := GetOrCreateRepositoryVersion(ctx, ownerID)
	if err != nil {
		return err
	}
	// remove old db files
	pfs, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		return err
	}
	for _, pf := range pfs {
		if pf.CompositeKey == distro && pf.Name == fmt.Sprintf("%s.db", distro) {
			if err := packages_service.DeletePackageFile(ctx, pf); err != nil {
				return err
			}
		}
	}

	db, err := flushDB(ctx, ownerID, distro)
	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	defer db.Close()
	// Create db signature cache
	_, err = db.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	sig, err := NewFileSign(ctx, ownerID, db)
	if err != nil {
		return err
	}
	defer sig.Close()
	_, err = db.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	for name, data := range map[string]*packages_module.HashedBuffer{
		fmt.Sprintf("%s.db", distro):     db,
		fmt.Sprintf("%s.db.sig", distro): sig,
	} {
		_, err = packages_service.AddFileToPackageVersionInternal(ctx, pv, &packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename:     name,
				CompositeKey: distro,
			},
			Creator:           user_model.NewGhostUser(),
			Data:              data,
			IsLead:            false,
			OverwriteExisting: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func flushDB(ctx context.Context, ownerID int64, distro string) (*packages_module.HashedBuffer, error) {
	pkgs, err := packages_model.GetPackagesByType(ctx, ownerID, packages_model.TypeArch)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, io.EOF
	}
	db, err := packages_module.NewHashedBuffer()
	if err != nil {
		return nil, err
	}
	gw := gzip.NewWriter(db)
	tw := tar.NewWriter(gw)
	errWrap := func(err error) error {
		return errors.Join(tw.Close(), gw.Close(), db.Close(), err)
	}
	count := 0
	for _, pkg := range pkgs {
		versions, err := packages_model.GetVersionsByPackageName(
			ctx, ownerID, packages_model.TypeArch, pkg.Name,
		)
		if err != nil {
			return nil, errWrap(err)
		}
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].CreatedUnix > versions[j].CreatedUnix
		})
		for _, ver := range versions {
			files, err := packages_model.GetFilesByVersionIDAndKey(ctx, ver.ID, distro)
			if err != nil {
				return nil, errWrap(err)
			}
			for _, pf := range files {
				pps, err := packages_model.GetPropertiesByName(
					ctx, packages_model.PropertyTypeFile, pf.ID, arch_module.PropertyDescription,
				)
				if err != nil {
					return nil, errWrap(err)
				}
				if len(pps) >= 1 {
					meta := []byte(pps[0].Value)
					header := &tar.Header{
						Name: pkg.Name + "-" + ver.Version + "/desc",
						Size: int64(len(meta)),
						Mode: int64(os.ModePerm),
					}
					if err = tw.WriteHeader(header); err != nil {
						return nil, errWrap(err)
					}
					if _, err := tw.Write(meta); err != nil {
						return nil, errWrap(err)
					}
					count++
					break
				}
			}
		}
	}
	defer gw.Close()
	defer tw.Close()
	if count == 0 {
		return nil, errors.Join(db.Close(), io.EOF)
	}
	return db, nil
}

// GetPackageFile Get data related to provided filename and distribution, for package files
// update download counter.
func GetPackageFile(ctx context.Context, distro, file string, ownerID int64) (io.ReadSeekCloser, error) {
	pf, err := getPackageFile(ctx, distro, file, ownerID)
	if err != nil {
		return nil, err
	}

	filestream, _, _, err := packages_service.GetPackageFileStream(ctx, pf)
	return filestream, err
}

// Ejects parameters required to get package file property from file name.
func getPackageFile(ctx context.Context, distro, file string, ownerID int64) (*packages_model.PackageFile, error) {
	var (
		splt    = strings.Split(file, "-")
		pkgname = strings.Join(splt[0:len(splt)-3], "-")
		vername = splt[len(splt)-3] + "-" + splt[len(splt)-2]
	)

	version, err := packages_model.GetVersionByNameAndVersion(ctx, ownerID, packages_model.TypeArch, pkgname, vername)
	if err != nil {
		return nil, err
	}

	pkgfile, err := packages_model.GetFileForVersionByName(ctx, version.ID, file, distro)
	if err != nil {
		return nil, err
	}
	return pkgfile, nil
}

func GetPackageDBFile(ctx context.Context, distro string, ownerID int64, signFile bool) (io.ReadSeekCloser, error) {
	pv, err := GetOrCreateRepositoryVersion(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	fileName := fmt.Sprintf("%s.db", distro)
	if signFile {
		fileName = fmt.Sprintf("%s.db.sig", distro)
	}
	file, err := packages_model.GetFileForVersionByName(ctx, pv.ID, fileName, distro)
	if err != nil {
		return nil, err
	}
	filestream, _, _, err := packages_service.GetPackageFileStream(ctx, file)
	return filestream, err
}

// GetOrCreateKeyPair gets or creates the PGP keys used to sign repository metadata files
func GetOrCreateKeyPair(ctx context.Context, ownerID int64) (string, string, error) {
	priv, err := user_model.GetSetting(ctx, ownerID, arch_module.SettingKeyPrivate)
	if err != nil && !errors.Is(err, util.ErrNotExist) {
		return "", "", err
	}

	pub, err := user_model.GetSetting(ctx, ownerID, arch_module.SettingKeyPublic)
	if err != nil && !errors.Is(err, util.ErrNotExist) {
		return "", "", err
	}

	if priv == "" || pub == "" {
		user, err := user_model.GetUserByID(ctx, ownerID)
		if err != nil && !errors.Is(err, util.ErrNotExist) {
			return "", "", err
		}

		priv, pub, err = generateKeypair(user.Name)
		if err != nil {
			return "", "", err
		}

		if err := user_model.SetUserSetting(ctx, ownerID, arch_module.SettingKeyPrivate, priv); err != nil {
			return "", "", err
		}

		if err := user_model.SetUserSetting(ctx, ownerID, arch_module.SettingKeyPublic, pub); err != nil {
			return "", "", err
		}
	}

	return priv, pub, nil
}

func generateKeypair(owner string) (string, string, error) {
	e, err := openpgp.NewEntity(
		owner,
		"Arch Package signature only",
		fmt.Sprintf("%s@noreply.%s", owner, setting.Packages.RegistryHost), &packet.Config{
			RSABits: 4096,
		})
	if err != nil {
		return "", "", err
	}

	var priv strings.Builder
	var pub strings.Builder

	w, err := armor.Encode(&priv, openpgp.PrivateKeyType, nil)
	if err != nil {
		return "", "", err
	}
	if err := e.SerializePrivate(w, nil); err != nil {
		return "", "", err
	}
	w.Close()

	w, err = armor.Encode(&pub, openpgp.PublicKeyType, nil)
	if err != nil {
		return "", "", err
	}
	if err := e.Serialize(w); err != nil {
		return "", "", err
	}
	w.Close()

	return priv.String(), pub.String(), nil
}
