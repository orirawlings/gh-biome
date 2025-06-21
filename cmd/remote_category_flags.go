package cmd

import (
	"strconv"

	"github.com/orirawlings/gh-biome/internal/biome"
	"github.com/spf13/pflag"
)

type remoteCategoryOptions struct {
	categories              map[biome.RemoteCategory]bool
	modified                bool
	fetchableCategoriesOnly bool
}

func newRemoteCategoryOptions(fetchableCategoriesOnly bool) *remoteCategoryOptions {
	o := &remoteCategoryOptions{
		fetchableCategoriesOnly: fetchableCategoriesOnly,
	}
	o.Reset()
	return o
}

func (o *remoteCategoryOptions) markModified() {
	if !o.modified {
		o.modified = true
		o.empty()
	}
}

func (o *remoteCategoryOptions) empty() {
	o.markModified()
	o.categories = make(map[biome.RemoteCategory]bool)
}

func (o *remoteCategoryOptions) all() []biome.RemoteCategory {
	if o.fetchableCategoriesOnly {
		return biome.FetchableRemoteCategories
	}
	return biome.AllRemoteCategories
}

func (o *remoteCategoryOptions) AddFlags(fs *pflag.FlagSet) {
	o.remoteCategoryValue(biome.Active).AddFlag(fs, "Include active remotes, repositories that are not archived, disabled, or locked in GitHub, and which are otherwise supported by this tool.")
	o.remoteCategoryValue(biome.Archived).AddFlag(fs, "Include remotes that are archived in GitHub, disabled from receiving new content. https://docs.github.com/en/repositories/archiving-a-github-repository")
	if !o.fetchableCategoriesOnly {
		o.remoteCategoryValue(biome.Disabled).AddFlag(fs, "Include remotes that are disabled in GitHub, unable to be updated. This seems to be a rare and undocumented condition for GitHub repositories. Disabled repositories cannot be fetched. Though discovered, these will not be added as actual git remotes on the biome.")
		o.remoteCategoryValue(biome.Locked).AddFlag(fs, "Include remotes that are locked in GitHub, disabled from any updates, usually because the repository has been migrated to a different git forge. Locked repositories cannot be fetched. Though discovered, these will not be added as actual git remotes on the biome. https://docs.github.com/en/migrations/overview/about-locked-repositories")
		o.remoteCategoryValue(biome.Unsupported).AddFlag(fs, "Include remotes that are currently unsupported by this tool. Unsupported remotes are skipped during remote configuration setup, but are still recorded in the configuration for reference.")
	}

	o.allRemoteCategoriesValue().AddFlag(fs, "Include all remotes, regardless of their status in GitHub.")
}

func (o *remoteCategoryOptions) Categories() []biome.RemoteCategory {
	categories := make([]biome.RemoteCategory, 0, len(o.categories))
	for category, ok := range o.categories {
		if ok {
			categories = append(categories, category)
		}
	}
	return categories
}

func (o *remoteCategoryOptions) Reset() {
	o.categories = map[biome.RemoteCategory]bool{
		biome.Active: true,
	}
	o.modified = false
}

func (o *remoteCategoryOptions) set(category biome.RemoteCategory, value bool) {
	o.markModified()
	o.categories[category] = value
}

func (o *remoteCategoryOptions) get(category biome.RemoteCategory) bool {
	return o.categories[category]
}

func (o *remoteCategoryOptions) remoteCategoryValue(category biome.RemoteCategory) *remoteCategoryValue {
	return &remoteCategoryValue{
		category: category,
		options:  o,
	}
}

func (o *remoteCategoryOptions) allRemoteCategoriesValue() *allRemoteCategoriesValue {
	return &allRemoteCategoriesValue{
		options: o,
	}
}

type remoteCategoryValue struct {
	category biome.RemoteCategory
	options  *remoteCategoryOptions
}

func (v *remoteCategoryValue) AddFlag(fs *pflag.FlagSet, description string) {
	f := fs.VarPF(v, string(v.category), "", description)
	f.NoOptDefVal = "true"
}

func (v *remoteCategoryValue) String() string {
	return strconv.FormatBool(v.options.get(v.category))
}

func (v *remoteCategoryValue) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	v.options.set(v.category, b)
	return nil
}

func (b *remoteCategoryValue) Type() string {
	return "bool"
}

func (b *remoteCategoryValue) IsBoolFlag() bool { return true }

type allRemoteCategoriesValue struct {
	options *remoteCategoryOptions
}

func (v *allRemoteCategoriesValue) AddFlag(fs *pflag.FlagSet, description string) {
	f := fs.VarPF(v, "all", "", description)
	f.NoOptDefVal = "true"
}

func (v *allRemoteCategoriesValue) String() string {
	value := true
	for _, category := range v.options.all() {
		value = value && v.options.get(category)
	}
	return strconv.FormatBool(value)
}

func (v *allRemoteCategoriesValue) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	if b {
		for _, category := range v.options.all() {
			v.options.set(category, true)
		}
	} else {
		v.options.empty()
	}
	return nil
}

func (b *allRemoteCategoriesValue) Type() string {
	return "bool"
}

func (b *allRemoteCategoriesValue) IsBoolFlag() bool { return true }
