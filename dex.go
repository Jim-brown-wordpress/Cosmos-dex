package dex

import (
    "github.com/cosmos/cosmos-sdk/codec"
    "github.com/cosmos/cosmos-sdk/types/errors"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
    RouterKey = "dex"
)

type Dex struct {
    dk              sdk.DecCoin
    address         sdk.AccAddress
    tokenContracts  map[string]sdk.AccAddress
    trades          []Trade
}

type Trade struct {
    takerAddr        sdk.AccAddress
    makerAddr        sdk.AccAddress
    takerAmount      sdk.DecCoin
    makerAmount      sdk.DecCoin
    takerTokenSymbol string
    makerTokenSymbol string
    timestamp        int64
}

func NewDex(ctx sdk.Context, dk sdk.DecCoin) Dex {
    // Create the dex address from the DK owner's address and a prefix
    address := sdk.AccAddress(crypto.AddressHash([]byte(dk.Owner.String() + "dex")))

    // Create the dex instance
    return Dex{
        dk:              dk,
        address:         address,
        tokenContracts:  make(map[string]sdk.AccAddress),
        trades:          []Trade{},
    }
}

func (d *Dex) AddToken(ctx sdk.Context, tokenSymbol string, tokenContract sdk.AccAddress) {
    d.tokenContracts[tokenSymbol] = tokenContract
}

func (d *Dex) CreateTrade(ctx sdk.Context, takerAddr sdk.AccAddress, takerAmount sdk.DecCoin, makerTokenSymbol string, makerAmount sdk.DecCoin) error {
    // Verify that the taker has enough funds
    takerBalance := bankKeeper.GetBalance(ctx, takerAddr, &takerAmount)
    if takerBalance.IsNegative() || takerBalance.LT(takerAmount.Amount) {
        return errors.Wrapf(errors.ErrInsufficientFunds, "insufficient funds for trade")
    }

    // Verify that the maker token is supported
    makerTokenAddr, exists := d.tokenContracts[makerTokenSymbol]
    if !exists {
        return errors.Wrapf(errors.ErrUnauthorized, "maker token not supported")
    }

    // Create the new trade
    newTrade := Trade{
        takerAddr:        takerAddr,
        makerAddr:        d.address,
        takerAmount:      takerAmount,
        makerAmount:      makerAmount,
        takerTokenSymbol: takerAmount.Denom,
        makerTokenSymbol: makerAmount.Denom,
        timestamp:        ctx.BlockTime().Unix(),
    }

    // Transfer the tokens from the taker to the dex
    err := bankKeeper.SendCoins(ctx, takerAddr, d.address, sdk.NewCoins(&takerAmount))
    if err != nil {
        return errors.Wrapf(err, "failed to transfer tokens from taker to dex")
    }

    // Transfer the tokens from the dex to the maker
    err = bankKeeper.SendCoins(ctx, d.address, makerTokenAddr, sdk.NewCoins(&makerAmount))
    if err != nil {
        return errors.Wrapf(err, "failed to transfer tokens from dex to maker")
    }

    // Add the new trade to the list of trades
    d.trades = append(d.trades, newTrade)

    // Emit a new trade event
    ctx.EventManager().EmitEvent(sdk.NewEvent(
        EventTypeTrade,
        sdk.NewAttribute(AttributeKeyTakerAddress, newTrade.takerAddr.String()),
        sdk.NewAttribute(AttributeKeyMakerAddress, newTrade.makerAddr.String()),
        sdk.NewAttribute(AttributeKeyTakerAmount, newTrade.takerAmount.String()),
        sdk.NewAttribute(AttributeKeyMakerAmount, newTrade.makerAmount.String()),
        sdk.NewAttribute(AttributeKeyTakerTokenSymbol, newTrade.takerTokenSymbol),
        sdk.NewAttribute(AttributeKeyMakerTokenSymbol, newTrade.makerTokenSymbol),
        sdk.NewAttribute(AttributeKeyTimestamp, fmt.Sprintf("%d", newTrade.timestamp)),
    ))

    return nil
}

func (d *Dex) GetTrades() []Trade {
    return d.trades
}

func (d *Dex) GetBalance(ctx sdk.Context, addr sdk.AccAddress, tokenSymbol string) sdk.DecCoin {
    tokenAddr, exists := d.tokenContracts[tokenSymbol]
    if !exists {
        return sdk.DecCoin{}
    }
    return bankKeeper.GetBalance(ctx, addr, tokenAddr)
}

func (d *Dex) GetAddress() sdk.AccAddress {
    return d.address
}

func (d *Dex) GetDenom() string {
    return d.dk.Denom
}

func (d *Dex) GetOwner() sdk.Acc
